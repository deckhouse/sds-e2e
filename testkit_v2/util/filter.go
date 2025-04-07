/*
Copyright 2025 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"regexp"
	"strings"
)

type Filter interface {
	Check(item any) bool
	Apply(item []any) []any
}

type Where interface {
	IsValid(string) bool
}

// "..." = Is
// "!..." = Not
// "%...%" = Like
// "!%...%" = NotLike
func CheckCondition(where any, val any) bool {
	if where == nil {
		return true
	}

	switch v := val.(type) {
	case string:
		switch w := where.(type) {
		case string:
			if len(w) >= 3 && w[0] == '%' && w[len(w)-1] == '%' {
				return WhereLike{w[1 : len(w)-1]}.IsValid(v)
			}
			if len(w) >= 4 && w[:2] == "!%" && w[len(w)-1] == '%' {
				return WhereNotLike{w[2 : len(w)-1]}.IsValid(v)
			}
			if len(w) >= 2 && w[0] == '!' {
				return WhereNotIn{w[1:]}.IsValid(v)
			}
			return WhereIn{w}.IsValid(v)
		case Where:
			return w.IsValid(v)
		default:
			Errf("Invalid filter type for string: %#v", w)
			return false
		}
	case bool:
		switch w := where.(type) {
		case bool:
			return w == v
		default:
			Errf("Invalid filter type for bool: %#v", w)
			return false
		}
	default:
		Errf("Invalid filter type: %#v", v)
		return false
	}

}

type WhereIn []string

func (f WhereIn) IsValid(val string) bool {
	for _, v := range f {
		if v == val {
			return true
		}
	}
	return false
}

type WhereNotIn []string

func (f WhereNotIn) IsValid(val string) bool {
	for _, v := range f {
		if v == val {
			return false
		}
	}
	return true
}

type WhereLike []string

func (f WhereLike) IsValid(val string) bool {
	for _, v := range f {
		if strings.Contains(val, v) {
			return true
		}
	}
	return false
}

type WhereNotLike []string

func (f WhereNotLike) IsValid(val string) bool {
	for _, v := range f {
		if strings.Contains(val, v) {
			return false
		}
	}
	return true
}

type WhereReg []string

func (f WhereReg) IsValid(val string) bool {
	for _, v := range f {
		match, err := regexp.MatchString(v, val)
		if err == nil && match {
			return true
		}
	}
	return false
}

type WhereNotReg []string

func (f WhereNotReg) IsValid(val string) bool {
	for _, v := range f {
		match, err := regexp.MatchString(v, val)
		if err == nil && match {
			return false
		}
	}
	return true
}
