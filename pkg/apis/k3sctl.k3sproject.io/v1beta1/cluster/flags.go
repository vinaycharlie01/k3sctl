package cluster

import (
	"fmt"
	"sort"
	"strings"
)

type Flags []string

func (f *Flags) Add(s string) {
	*f = append(*f, s)
}

func (f *Flags) AddOrReplace(key, value string) {
	prefix := key + "="
	for i, flag := range *f {
		if strings.HasPrefix(flag, prefix) || flag == key {
			if value == "" {
				(*f)[i] = key
			} else {
				(*f)[i] = prefix + value
			}
			return
		}
	}
	if value == "" {
		f.Add(key)
	} else {
		f.Add(prefix + value)
	}
}

func (f *Flags) Delete(key string) {
	prefix := key + "="
	result := make(Flags, 0, len(*f))
	for _, flag := range *f {
		if flag == key || strings.HasPrefix(flag, prefix) {
			continue
		}
		result = append(result, flag)
	}
	*f = result
}

func (f Flags) GetValue(key string) (string, bool) {
	prefix := key + "="
	for _, flag := range f {
		if strings.HasPrefix(flag, prefix) {
			return strings.TrimPrefix(flag, prefix), true
		}
		if flag == key {
			return "", true
		}
	}
	return "", false
}

func (f Flags) Include(key string) bool {
	_, found := f.GetValue(key)
	return found
}

func (f Flags) Join() string {
	return strings.Join(f, " ")
}

func (f Flags) Sorted() Flags {
	result := make(Flags, len(f))
	copy(result, f)
	sort.Strings(result)
	return result
}

func (f Flags) Equals(other Flags) bool {
	if len(f) != len(other) {
		return false
	}
	a := f.Sorted()
	b := other.Sorted()
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (f Flags) String() string {
	return f.Join()
}

func (f Flags) Map() map[string]string {
	result := make(map[string]string, len(f))
	for _, flag := range f {
		parts := strings.SplitN(flag, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		} else {
			result[parts[0]] = ""
		}
	}
	return result
}

func FlagsFromMap(m map[string]string) Flags {
	f := make(Flags, 0, len(m))
	for k, v := range m {
		if v == "" {
			f = append(f, k)
		} else {
			f = append(f, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return f.Sorted()
}
