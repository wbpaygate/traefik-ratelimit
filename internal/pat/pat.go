package pat

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type Pat int

func Appendpat(ipat [][]Pat, ipt []Pat) [][]Pat {
	if ipt == nil {
		return ipat
	}
	for _, tipt := range ipat {
		if slices.Equal(tipt, ipt) {
			return ipat
		}
	}
	return append(ipat, ipt)
}

func Preparepat(ipt []Pat, s string) (string, bool) {
	//	fmt.Println("prep", s)
	ss := strings.Split(s, "/")
	r := make([]string, 0, len(ipt))
	for _, i := range ipt {
		j := i
		if i < 0 {
			j = Pat(len(ss)) + i
		}
		if j > Pat(len(ss))-1 || j < 0 {
			return "", false
		}
		//		fmt.Println("prep", i, len(ss), j)
		r = append(r, strconv.Itoa(int(i))+":"+ss[j])
	}
	return strings.Join(r, "/"), true
}

func Compilepat(s string) (string, []Pat, error) {
	if len(strings.TrimSpace(s)) == 0 {
		return "", nil, nil
	}
	f := 0
	fl := false
	ss := strings.Split(s, "/")
	r := make([]string, 0, len(ss))
	ri := make([]Pat, 0, len(ss))
	for i, s := range ss {
		switch s {
		case "**":
			fl = true
			if f > 0 {
				return "", nil, fmt.Errorf("bad pattern")
			}
			f = i + 1
		case "*", "":
		default:
			r = append(r, s)
			ri = append(ri, Pat(i))
		}
	}
	for i := range r {
		if ri[i] >= Pat(f) && fl {
			ri[i] = ri[i] - ri[len(ri)-1] - 1
		}
		r[i] = strconv.Itoa(int(ri[i])) + ":" + r[i]
	}
	return strings.Join(r, "/"), ri, nil
}