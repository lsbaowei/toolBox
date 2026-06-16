package utils_cache_version

import (
	"errors"
	"hash/fnv"
	"math"
	"sync"
	"time"
)

const bucketSize uint64 = 10000

var (
	// ErrInvalidReleaseDuration 表示释放窗口未配置或配置为非正数。
	ErrInvalidReleaseDuration = errors.New("release duration must be positive")
	// ErrEmptyVersion 表示第三方版本为空。
	ErrEmptyVersion = errors.New("version must not be empty")
	// ErrEmptyBusinessID 表示业务标识为空，无法进行确定性灰度。
	ErrEmptyBusinessID = errors.New("business id must not be empty")
)

// Config 定义缓存版本渐进释放配置。
type Config struct {
	// ReleaseDuration 是从稳定版本逐步释放到目标版本的窗口。
	ReleaseDuration time.Duration
	// Now 在 SelectOptions.Now 未提供时作为当前时间来源；为空时使用 time.Now。
	Now func() time.Time
}

// SelectOptions 是单次版本选择的输入。
type SelectOptions struct {
	// BusinessID 是稳定且分布足够均匀的业务标识，如用户 ID、租户 ID 或资源 ID。
	BusinessID string
	// Version 是第三方当前返回的最新版本。
	Version string
	// Now 是本次选择使用的当前时间；零值时使用 Config.Now 或 time.Now。
	Now time.Time
}

// Result 描述本次应使用的缓存版本及释放状态。
type Result struct {
	// Version 是调用方应拼接到缓存 key 中的版本。
	Version string
	// StableVersion 是当前已完成释放的稳定版本。
	StableVersion string
	// TargetVersion 是正在释放的目标版本；没有活跃释放时为空。
	TargetVersion string
	// InRelease 表示当前是否存在活跃释放阶段。
	InRelease bool
	// Released 表示本次业务标识是否命中目标版本。
	Released bool
	// Progress 表示活跃释放阶段进度，范围为 [0, 1]。
	Progress float64
}

type releaseStage struct {
	baseVersion   string
	targetVersion string
	start         time.Time
	sticky        map[string]struct{}
}

// Manager 维护单进程内的版本释放状态。
type Manager struct {
	mu            sync.RWMutex
	cfg           Config
	stableVersion string
	active        *releaseStage
}

// New 创建缓存版本释放管理器。配置会在 SelectVersion 时校验。
func New(cfg Config) *Manager {
	return &Manager{cfg: cfg}
}

// SelectVersion 根据业务标识、第三方当前版本和释放进度返回本次应使用的缓存版本。
func (m *Manager) SelectVersion(opt SelectOptions) (Result, error) {
	if m == nil {
		m = New(Config{})
	}

	// 阶段 1：校验与共享状态无关的基础输入，避免无效请求进入临界区。
	if m.cfg.ReleaseDuration <= 0 {
		return Result{}, ErrInvalidReleaseDuration
	}
	if opt.Version == "" {
		return Result{}, ErrEmptyVersion
	}

	// 阶段 2：固定稳定版本是现实业务中的最高频路径。
	// 该路径不修改共享状态，不需要 BusinessID、当前时间或写锁。
	m.mu.RLock()
	if m.stableVersion != "" && m.active == nil && opt.Version == m.stableVersion {
		stableVersion := m.stableVersion
		m.mu.RUnlock()
		return Result{Version: stableVersion, StableVersion: stableVersion}, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// 阶段 3：首次观察到的第三方版本直接成为稳定版本，不启动灰度释放。
	if m.stableVersion == "" {
		m.stableVersion = opt.Version
		return Result{Version: opt.Version, StableVersion: opt.Version}, nil
	}

	// 阶段 4：写锁内二次检查稳定版本快速路径。
	// 读锁释放到写锁获取之间状态可能变化，必须重新判断一次。
	if opt.Version == m.stableVersion && m.active == nil {
		return Result{Version: m.stableVersion, StableVersion: m.stableVersion}, nil
	}
	if opt.BusinessID == "" {
		return Result{}, ErrEmptyBusinessID
	}

	// 阶段 5：确定本次选择使用的时间；只有释放路径才需要计算时间。
	now := m.currentTime(opt.Now)

	// 阶段 6：观察第三方最新版本，必要时开启释放阶段或更新活跃目标版本。
	m.observeVersion(opt.Version, now)

	// 阶段 7：释放窗口完成后，把活跃目标提升为稳定版本并清理阶段粘性状态。
	m.completeRelease(now)

	if m.active == nil {
		return Result{Version: m.stableVersion, StableVersion: m.stableVersion}, nil
	}

	// 阶段 8：根据释放进度、确定性分桶和粘性记录选择本次缓存版本。
	progress := m.releaseProgress(now)
	if _, ok := m.active.sticky[opt.BusinessID]; ok {
		return m.activeResult(m.active.targetVersion, true, progress), nil
	}
	if bucket(opt.BusinessID, m.active.targetVersion) < progressThreshold(progress) {
		m.active.sticky[opt.BusinessID] = struct{}{}
		return m.activeResult(m.active.targetVersion, true, progress), nil
	}
	return m.activeResult(m.active.baseVersion, false, progress), nil
}

func (m *Manager) observeVersion(version string, now time.Time) {
	if version == m.stableVersion {
		return
	}
	if m.active == nil {
		m.active = &releaseStage{
			baseVersion:   m.stableVersion,
			targetVersion: version,
			start:         now,
			sticky:        make(map[string]struct{}),
		}
		return
	}
	if version != m.active.targetVersion {
		m.active.targetVersion = version
	}
}

func (m *Manager) completeRelease(now time.Time) {
	if m.active == nil || m.cfg.ReleaseDuration <= 0 {
		return
	}
	if now.Sub(m.active.start) < m.cfg.ReleaseDuration {
		return
	}
	m.stableVersion = m.active.targetVersion
	m.active = nil
}

func (m *Manager) currentTime(explicit time.Time) time.Time {
	if !explicit.IsZero() {
		return explicit
	}
	if m != nil && m.cfg.Now != nil {
		return m.cfg.Now()
	}
	return time.Now()
}

func (m *Manager) releaseProgress(now time.Time) float64 {
	if m.active == nil || m.cfg.ReleaseDuration <= 0 {
		return 0
	}
	elapsed := now.Sub(m.active.start)
	if elapsed <= 0 {
		return 0
	}
	if elapsed >= m.cfg.ReleaseDuration {
		return 1
	}
	return float64(elapsed) / float64(m.cfg.ReleaseDuration)
}

func (m *Manager) activeResult(version string, released bool, progress float64) Result {
	return Result{
		Version:       version,
		StableVersion: m.stableVersion,
		TargetVersion: m.active.targetVersion,
		InRelease:     true,
		Released:      released,
		Progress:      progress,
	}
}

func bucket(businessID, version string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(businessID))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(version))
	return h.Sum64() % bucketSize
}

func progressThreshold(progress float64) uint64 {
	switch {
	case progress <= 0:
		return 0
	case progress >= 1:
		return bucketSize
	default:
		return uint64(math.Floor(progress * float64(bucketSize)))
	}
}
