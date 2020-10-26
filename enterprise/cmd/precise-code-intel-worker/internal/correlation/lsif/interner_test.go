package lsif

import "testing"

func TestInterner(t *testing.T) {
	testCases := [][][]byte{
		{
			[]byte(`1`),
			[]byte(`2`),
			[]byte(`3`),
			[]byte(`4`),
			[]byte(`5`),
			[]byte(`100`),
			[]byte(`200`),
			[]byte(`300`),
			[]byte(`400`),
			[]byte(`500`),
		},
		{
			[]byte(`"1"`),
			[]byte(`"2"`),
			[]byte(`"3"`),
			[]byte(`"4"`),
			[]byte(`"5"`),
			[]byte(`"100"`),
			[]byte(`"200"`),
			[]byte(`"300"`),
			[]byte(`"400"`),
			[]byte(`"500"`),
		},
		{
			[]byte(`"17f5d4ea-b851-4189-9de7-736002d52d05"`),
			[]byte(`"dc916a1f-c34b-45f0-80ce-e1fc00c019d5"`),
			[]byte(`"46a0ca88-4abc-4180-bc52-3745c3414b6a"`),
			[]byte(`"ae581041-3ed5-444f-8dab-d1e2363cd936"`),
			[]byte(`"da74139d-1403-4e76-b5be-3fe9ce04ecf8"`),
		},
		{
			[]byte(`"rectangle"`),
			[]byte(`"america"`),
			[]byte(`"megaphone"`),
			[]byte(`"monday"`),
			[]byte(`"the next word"`),
		},
	}

	for _, bytes := range testCases {
		interner := NewInterner()
		returned := map[string]int{}

		for _, b := range bytes {
			v, err := interner.Intern(b)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if _, ok := returned[string(b)]; ok {
				t.Fatalf("duplicate id")
			}

			returned[string(b)] = v
		}

		for _, b := range bytes {
			v, err := interner.Intern(b)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if v != returned[string(b)] {
				t.Fatalf("id does not match existing value")
			}
		}
	}
}
