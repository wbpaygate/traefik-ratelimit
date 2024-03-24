package pat

import (
	//	"fmt"
	"slices"
	"testing"
)

func Test_compilepat(t *testing.T) {
	cases := []struct {
		name   string
		s      string
		resp   string
		resipt []Pat
		reserr bool
	}{

		{
			name:   "t1",
			s:      "/",
			resp:   "1:",
			resipt: []Pat{Pat{v: 1}},
			reserr: false,
		},
		{
			name:   "t2",
			s:      "/$",
			resp:   "1:/1:$",
			resipt: []Pat{Pat{v: 1}, Pat{t: 2, v: 1}},
			reserr: false,
		},

		{
			name:   "t3",
			s:      "/aa$",
			resp:   "1:aa/1:$",
			resipt: []Pat{Pat{v: 1}, Pat{t: 2, v: 1}},
			reserr: false,
		},
		{
			name:   "t4",
			s:      "/aa/$",
			resp:   "1:aa/2:/2:$",
			resipt: []Pat{Pat{v: 1}, Pat{v: 2}, Pat{t: 2, v: 2}},
			reserr: false,
		},

		{
			name:   "t5",
			s:      "/aa",
			resp:   "1:aa",
			resipt: []Pat{Pat{v: 1}},
			reserr: false,
		},

		{
			name:   "t6",
			s:      "/**/aa",
			resp:   "-1:aa",
			resipt: []Pat{Pat{v: -1}},
			reserr: false,
		},
		{
			name:   "t7",
			s:      "**/aa",
			resp:   "-1:aa",
			resipt: []Pat{Pat{v: -1}},
			reserr: false,
		},

		{
			name:   "t8",
			s:      "/a/**/aa",
			resp:   "1:a/-1:aa",
			resipt: []Pat{Pat{v: 1}, Pat{v: -1}},
			reserr: false,
		},
		{
			name:   "t9",
			s:      "/a/**/aa$",
			reserr: true,
		},

		{
			name:   "t10",
			s:      "/a/*/b/**/aa",
			resp:   "1:a/2:*/3:b/-1:aa",
			resipt: []Pat{Pat{v: 1}, Pat{t: 1, v: 2}, Pat{v: 3}, Pat{v: -1}},
			reserr: false,
		},

		{
			name:   "t11",
			s:      "/a/*/b/**/a/*/b",
			resp:   "1:a/2:*/3:b/-3:a/-2:*/-1:b",
			resipt: []Pat{Pat{v: 1}, Pat{t: 1, v: 2}, Pat{v: 3}, Pat{v: -3}, Pat{t: 1, v: -2}, Pat{v: -1}},
			reserr: false,
		},

		{
			name:   "t12",
			s:      "/a/*/b/*/a/*/b/",
			resp:   "1:a/2:*/3:b/4:*/5:a/6:*/7:b/8:",
			resipt: []Pat{Pat{v: 1}, Pat{t: 1, v: 2}, Pat{v: 3}, Pat{t: 1, v: 4}, Pat{v: 5}, Pat{t: 1, v: 6}, Pat{v: 7}, Pat{v: 8}},
			reserr: false,
		},

		{
			name:   "t13",
			s:      "/a/*/b/**/a/*/b/",
			resp:   "1:a/2:*/3:b/-4:a/-3:*/-2:b/-1:",
			resipt: []Pat{Pat{v: 1}, Pat{t: 1, v: 2}, Pat{v: 3}, Pat{v: -4}, Pat{t: 1, v: -3}, Pat{v: -2}, Pat{v: -1}},
			reserr: false,
		},

		{
			name:   "t14",
			s:      "/a/**/b/**/aa",
			reserr: true,
		},
		{
			name:   "t10",
			s:      "/a/*/b/*/aa/",
			resp:   "1:a/2:*/3:b/4:*/5:aa/6:",
			resipt: []Pat{Pat{v: 1}, Pat{t: 1, v: 2}, Pat{v: 3}, Pat{t: 1, v: 4}, Pat{v: 5}, Pat{v: 6}},
			reserr: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resp, resipt, err := Compilepat(tc.s)
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
