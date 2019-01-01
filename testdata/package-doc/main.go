// LICENSE TEXT

// Package main is an example package.
package main

// superfluous comment

import (
	"fmt"
	mth "math" //@pkgdoc("th", "Package math provides"), pkgdoc("ath", "Package math provides")
)

func main() {
	fmt.Println(mth.IsNaN(1.23)) //@pkgdoc("th.IsNaN", "Package math provides")
	fmt.Println("Goodbye")       //@pkgdoc("mt.", "\tPackage fmt implements formatted I/O")
}
