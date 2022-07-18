package utils

import (
	_ "image/png"
	"math/rand"
)

func GenerateRandomNumber(rnd *rand.Rand, min, max int) int {
	ns := make([]int, 0)

	if max == 0 {
		return 0
	}

	for i := 0; i < 100; i++ {
		//fmt.Println("max")
		//fmt.Println(max)
		n := rnd.Intn(max)
		ns = append(ns, n)
	}
	n2 := rnd.Intn(len(ns))

	return ns[n2]
}

/*

	rnd = rand.New(src)
	p := GenerateRandomNumber(rnd, 1, 100000)
	fmt.Println(p)

*/
