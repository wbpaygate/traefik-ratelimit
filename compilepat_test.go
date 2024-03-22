package traefik_ratelimit

import (
//	"fmt"
	"slices"
	"testing"
)

func Test_compilepat(t *testing.T) {
	cases := []struct {
		name string
		s    string
		resp   string
		resipt []int
		reserr bool
	} {
		{
			name:   "t1",
			s:      "/",
			resp:   "",
			resipt: []int{},
			reserr: false,
		},

		{
			name:   "t2",
			s:      "/aa",
			resp:   "1:aa",
			resipt: []int{1},
			reserr: false,
		},

		{
			name:   "t1",
			s:      "/**/aa",
			resp:   "-1:aa",
			resipt: []int{-1},
			reserr: false,
		},

		{
			name:   "t1",
			s:      "/a/**/aa",
			resp:   "1:a/-1:aa",
			resipt: []int{1,-1},
			reserr: false,
		},

		{
			name:   "t1",
			s:      "/a/*/b/**/aa",
			resp:   "1:a/3:b/-1:aa",
			resipt: []int{1,3,-1},
			reserr: false,
		},

		{
			name:   "t1",
			s:      "/a/*/b/**/a/*/b",
			resp:   "1:a/3:b/-3:a/-1:b",
			resipt: []int{1,3,-3,-1},
			reserr: false,
		},

		{
			name:   "t1",
			s:      "/a/**/b/**/aa",
			reserr: true,
		},

	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resp, resipt, err := compilepat(tc.s)
			if err != nil && !tc.reserr {
				t.Errorf("compilepat %s not expected error %v", tc.name, err)
				return
			}
			if err == nil && tc.reserr {
				t.Errorf("compilepat %s expect error but have result %q %v", tc.name, resp, resipt)
				return
			}
			if resp != tc.resp || !slices.Equal(resipt, tc.resipt) {
				t.Errorf("compilepat %s expect %q %q , %v %v", tc.name, resp, tc.resp, resipt, tc.resipt)
				return
			}
		})
	}
}


