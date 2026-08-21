package main

import (
	"fmt"
	"image/png"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: analyze <png>")
		return
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		panic(err)
	}
	b := img.Bounds()
	fmt.Printf("size=%dx%d type=%T\n", b.Dx(), b.Dy(), img)

	type cnt struct{ r, g, bl, n int }
	order := map[string]*cnt{}
	orderList := []string{}
	tot := b.Dx() * b.Dy()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			r >>= 8
			g >>= 8
			bl >>= 8
			key := fmt.Sprintf("%d,%d,%d", r, g, bl)
			if c, ok := order[key]; ok {
				c.n++
			} else {
				order[key] = &cnt{int(r), int(g), int(bl), 1}
				orderList = append(orderList, key)
			}
		}
	}
	// top 12
	type kv struct {
		k string
		c *cnt
	}
	list := make([]kv, 0, len(orderList))
	for _, k := range orderList {
		list = append(list, kv{k, order[k]})
	}
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j-1].c.n < list[j].c.n; j-- {
			list[j-1], list[j] = list[j], list[j-1]
		}
	}
	fmt.Println("uniqColors=", len(list))
	for i := 0; i < len(list) && i < 14; i++ {
		c := list[i].c
		fmt.Printf("color (%d,%d,%d)  count=%d  %.2f%%\n", c.r, c.g, c.bl, c.n, 100*float64(c.n)/float64(tot))
	}
}