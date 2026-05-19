package utils_random_test

import (
	"fmt"

	"github.com/lsbaowei/toolBox/utils_random"
	"github.com/lsbaowei/toolBox/utils_time"
)

func ExampleSecureIntn() {
	n, err := utils_random.SecureIntn(100)
	if err != nil {
		fmt.Println("error")
		return
	}
	fmt.Println(n >= 0)
	// Output: true
}

func ExampleDateTime_Random() {
	d := utils_time.New(nil)
	n := d.Random(100)
	fmt.Println(n >= 0)
	// Output: true
}

func ExampleRandUtil_Intn() {
	ru := utils_random.New()
	n := ru.Intn(100)
	fmt.Println(n >= 0)
	// Output: true
}
