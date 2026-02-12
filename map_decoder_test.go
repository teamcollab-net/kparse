package kparse

import (
	"encoding/json"
	"testing"

	tt "github.com/teamcollab-net/kparse/internal/testtools"
)

func TestMapTagDecoder(t *testing.T) {
	t.Run("basic parsing", func(t *testing.T) {
		tests := []struct {
			desc               string
			input              map[string]LazyDecoder
			target             any
			expected           any
			expectErrToContain []string
		}{
			{
				desc: "should work for valid structs",
				input: map[string]LazyDecoder{
					"id":       testDecoder(42),
					"username": testDecoder("fakeUsername"),
					"address": testDecoder(map[string]any{
						"street":  "fakeStreet",
						"city":    "fakeCity",
						"country": "fakeCountry",
					}),
				},
				target: &struct {
					ID       int    `map:"id"`
					Username string `map:"username"`
					Address  struct {
						Street  string `map:"street"`
						City    string `map:"city"`
						Country string `map:"country"`
					} `map:"address"`
				}{},
				expected: &struct {
					ID       int    `map:"id"`
					Username string `map:"username"`
					Address  struct {
						Street  string `map:"street"`
						City    string `map:"city"`
						Country string `map:"country"`
					} `map:"address"`
				}{
					ID:       42,
					Username: "fakeUsername",
					Address: struct {
						Street  string `map:"street"`
						City    string `map:"city"`
						Country string `map:"country"`
					}{
						Street:  "fakeStreet",
						City:    "fakeCity",
						Country: "fakeCountry",
					},
				},
			},
			{
				desc: "should work for structs with string slices",
				input: map[string]LazyDecoder{
					"id":    testDecoder(42),
					"slice": testDecoder([]string{"fakeUser1", "fakeUser2"}),
				},
				target: &struct {
					ID    int      `map:"id"`
					Slice []string `map:"slice"`
				}{},
				expected: &struct {
					ID    int      `map:"id"`
					Slice []string `map:"slice"`
				}{
					ID: 42,
					Slice: []string{
						"fakeUser1",
						"fakeUser2",
					},
				},
			},
			{
				desc: "should work for structs with struct slices",
				input: map[string]LazyDecoder{
					"id": testDecoder(42),
					"slice": testDecoder([]map[string]any{
						{"name": "fakeUser1"},
						{"name": "fakeUser2"},
					}),
				},
				target: &struct {
					ID    int `map:"id"`
					Slice []struct {
						Name string `map:"name"`
					} `map:"slice"`
				}{},
				expected: &struct {
					ID    int `map:"id"`
					Slice []struct {
						Name string `map:"name"`
					} `map:"slice"`
				}{
					ID: 42,
					Slice: []struct {
						Name string `map:"name"`
					}{
						{Name: "fakeUser1"},
						{Name: "fakeUser2"},
					},
				},
			},
			{
				desc: "should return error if we try to save something that is not a map into a nested struct",
				input: map[string]LazyDecoder{
					"id":      testDecoder(42),
					"address": testDecoder("notAMap"),
				},
				target: &struct {
					ID      int `map:"id"`
					Address struct {
						Street string `map:"street"`
					} `map:"address"`
				}{},
				expectErrToContain: []string{
					"string",
					"Address",
					"Street",
				},
			},
		}

		for _, test := range tests {
			t.Run(test.desc, func(t *testing.T) {
				err := parseFromMap("map", test.target, test.input)
				if test.expectErrToContain != nil {
					tt.AssertErrContains(t, err, test.expectErrToContain...)
					return
				}
				tt.AssertNoErr(t, err)

				tt.AssertEqual(t, test.target, test.expected)
			})
		}
	})

	t.Run("validations", func(t *testing.T) {
		t.Run("using the required validation", func(t *testing.T) {
			t.Run("should ignore missing fields if they are not required", func(t *testing.T) {
				var user struct {
					ID       int    `map:"id"`
					Username string `map:"username"`
					Address  struct {
						Street  string `map:"street"`
						City    string `map:"city"`
						Country string `map:"country"`
					} `map:"address"`

					OptionalStruct struct {
						ID int `map:"id"`
					} `map:"optional_struct"`
				}
				// These three should still be present after the parsing:
				user.OptionalStruct.ID = 42
				user.Username = "presetUsername"
				user.Address.Street = "presetStreet"

				// These two should be overwritten by the parser:
				user.ID = 43
				user.Address.Country = "presetCountry"

				err := parseFromMap("map", &user, map[string]LazyDecoder{
					"id": testDecoder(44),
					"address": testDecoder(map[string]any{
						"city":    "fakeCity",
						"country": "fakeCountry",
					}),
				})
				tt.AssertNoErr(t, err)

				tt.AssertEqual(t, user.ID, 44)
				tt.AssertEqual(t, user.Username, "presetUsername")
				tt.AssertEqual(t, user.Address.Street, "presetStreet")
				tt.AssertEqual(t, user.Address.City, "fakeCity")
				tt.AssertEqual(t, user.Address.Country, "fakeCountry")
				tt.AssertEqual(t, user.OptionalStruct.ID, 42)
			})

			t.Run("should return error for missing fields if they are required", func(t *testing.T) {
				tests := []struct {
					desc               string
					input              map[string]LazyDecoder
					expectErrToContain []string
				}{
					{
						desc: "required field missing on root map",
						input: map[string]LazyDecoder{
							"id": testDecoder(42),
							"address": testDecoder(map[string]any{
								"street":  "fakeStreet",
								"city":    "fakeCity",
								"country": "fakeCountry",
							}),
						},
						expectErrToContain: []string{"missing", "required", "username"},
					},
					{
						desc: "required field missing on nested map",
						input: map[string]LazyDecoder{
							"id":       testDecoder(42),
							"username": testDecoder("fakeUsername"),
							"address": testDecoder(map[string]any{
								"city":    "fakeCity",
								"country": "fakeCountry",
							}),
						},
						expectErrToContain: []string{"missing", "required", "street"},
					},
					{
						desc: "required field missing is a map",
						input: map[string]LazyDecoder{
							"id":       testDecoder(42),
							"username": testDecoder("fakeUsername"),
						},
						expectErrToContain: []string{"missing", "required", "address"},
					},
				}

				for _, test := range tests {
					t.Run(test.desc, func(t *testing.T) {
						var user struct {
							ID       int    `map:"id"`
							Username string `map:"username" validate:"required"`
							Address  struct {
								Street  string `map:"street" validate:"required"`
								City    string `map:"city"`
								Country string `map:"country"`
							} `map:"address" validate:"required"`
						}
						err := parseFromMap("map", &user, test.input)

						tt.AssertErrContains(t, err, test.expectErrToContain...)
					})
				}
			})

			t.Run("should return error if the validation is misspelled", func(t *testing.T) {
				var user struct {
					ID       int    `map:"id"`
					Username string `map:"username" validate:"not_required"`
				}
				err := parseFromMap("map", &user, map[string]LazyDecoder{
					"id":       testDecoder(42),
					"username": testDecoder("fakeUsername"),
				})

				tt.AssertErrContains(t, err, "validation", "not_required")
			})

			t.Run("should not return error if the required field is empty but has a default value", func(t *testing.T) {
				var user struct {
					ID       int    `map:"id"`
					Username string `map:"username" validate:"required" default:"defaultUsername"`
				}
				err := parseFromMap("map", &user, map[string]LazyDecoder{
					"id": testDecoder(42),
				})
				tt.AssertNoErr(t, err)

				tt.AssertEqual(t, user.ID, 42)
				tt.AssertEqual(t, user.Username, "defaultUsername")
			})
		})

		t.Run("logical validations", func(t *testing.T) {
			tests := []struct {
				desc               string
				structPtr          any
				sourceMap          map[string]LazyDecoder
				expectErrToContain []string
			}{
				{
					desc: "should allow values in range",
					structPtr: &struct {
						Salary     int `map:"salary" validate:">1000"`
						AgeInYears int `map:"age" validate:"<100"`
						HeightInCm int `map:"height" validate:">140,<230"`
					}{},
					sourceMap: map[string]LazyDecoder{
						"salary": testDecoder(2000),
						"age":    testDecoder(45),
						"height": testDecoder(179),
					},
				},
				{
					desc: "should block if value below minimum",
					structPtr: &struct {
						ValidBefore int `map:"before" validate:"<100"`
						BelowMin    int `map:"belowMin" validate:">1000"`
						ValidAfter  int `map:"after" validate:">140,<230"`
					}{},
					sourceMap: map[string]LazyDecoder{
						"before":   testDecoder(45),
						"belowMin": testDecoder(500),
						"after":    testDecoder(178),
					},
					expectErrToContain: []string{"BelowMin", ">", "500", "1000"},
				},
				{
					desc: "should block if value above maximum",
					structPtr: &struct {
						ValidBefore int `map:"before" validate:"<100"`
						AboveMax    int `map:"aboveMax" validate:"<100"`
						ValidAfter  int `map:"after" validate:">140,<230"`
					}{},
					sourceMap: map[string]LazyDecoder{
						"before":   testDecoder(45),
						"aboveMax": testDecoder(500),
						"after":    testDecoder(178),
					},
					expectErrToContain: []string{"AboveMax", "<", "500", "100"},
				},
				{
					desc: "should not fail the validation if value is missing",
					structPtr: &struct {
						ValidBefore  int `map:"before" validate:"<100"`
						MissingValue int `map:"missing" validate:"<100"`
						ValidAfter   int `map:"after" validate:">140,<230"`
					}{},
					sourceMap: map[string]LazyDecoder{
						"before": testDecoder(45),
						"after":  testDecoder(178),
					},
				},
				{
					desc: "should not conflict with the required validation when field when all is valid",
					structPtr: &struct {
						ValidBefore int `map:"before" validate:"<100"`
						Required    int `map:"required" validate:"required,<100"`
						ValidAfter  int `map:"after" validate:">140,<230"`
					}{},
					sourceMap: map[string]LazyDecoder{
						"before":   testDecoder(45),
						"required": testDecoder(50),
						"after":    testDecoder(178),
					},
				},
				{
					desc: "should not conflict with the required validation when field is missing",
					structPtr: &struct {
						ValidBefore int `map:"before" validate:"<100"`
						Required    int `map:"required" validate:"required,<100"`
						ValidAfter  int `map:"after" validate:">140,<230"`
					}{},
					sourceMap: map[string]LazyDecoder{
						"before": testDecoder(45),
						"after":  testDecoder(178),
					},
					expectErrToContain: []string{"missing", "required"},
				},
				{
					desc: "should not conflict with the required validation when field is not in range",
					structPtr: &struct {
						ValidBefore int `map:"before" validate:"<100"`
						Required    int `map:"required" validate:"required,<100"`
						ValidAfter  int `map:"after" validate:">140,<230"`
					}{},
					sourceMap: map[string]LazyDecoder{
						"before":   testDecoder(45),
						"required": testDecoder(120), // not in range
						"after":    testDecoder(178),
					},
					expectErrToContain: []string{"Required", "120", "<", "100"},
				},
				{
					desc: "should work for negative fields",
					structPtr: &struct {
						ValidBefore int `map:"before" validate:"<100"`
						MinNegative int `map:"minNegative" validate:">-10"`
						MaxNegative int `map:"maxNegative" validate:"<-10"`
						ValidAfter  int `map:"after" validate:">140,<230"`
					}{},
					sourceMap: map[string]LazyDecoder{
						"before":      testDecoder(45),
						"minNegative": testDecoder(-5),
						"maxNegative": testDecoder(-15),
						"after":       testDecoder(178),
					},
				},
				{
					desc: "should fail for min out of range on a negative field",
					structPtr: &struct {
						ValidBefore int `map:"before" validate:"<100"`
						Negative    int `map:"negative" validate:">-10"`
						ValidAfter  int `map:"after" validate:">140,<230"`
					}{},
					sourceMap: map[string]LazyDecoder{
						"before":   testDecoder(45),
						"negative": testDecoder(-15),
						"after":    testDecoder(178),
					},
					expectErrToContain: []string{"Negative", "-15", ">", "-10"},
				},
				{
					desc: "should fail for max out of range on a negative field",
					structPtr: &struct {
						ValidBefore int `map:"before" validate:"<100"`
						Negative    int `map:"negative" validate:"<-10"`
						ValidAfter  int `map:"after" validate:">140,<230"`
					}{},
					sourceMap: map[string]LazyDecoder{
						"before":   testDecoder(45),
						"negative": testDecoder(-5),
						"after":    testDecoder(178),
					},
					expectErrToContain: []string{"Negative", "-5", "<", "-10"},
				},
				{
					desc: "no error for equal validation",
					structPtr: &struct {
						V int `map:"v" validate:"=3"`
					}{},
					sourceMap: map[string]LazyDecoder{"v": testDecoder(3)},
				},
				{
					desc: "error for equal validation",
					structPtr: &struct {
						V int `map:"v" validate:"=4"`
					}{},
					sourceMap:          map[string]LazyDecoder{"v": testDecoder(3)},
					expectErrToContain: []string{"V", "3", "=", "4"},
				},
				{
					desc: "min/max should work for different types of numbers",
					structPtr: &struct {
						Int   int   `map:"int" validate:">0,<100"`
						Int8  int8  `map:"int8" validate:">0,<100"`
						Int16 int16 `map:"int16" validate:">0,<100"`
						Int32 int32 `map:"int32" validate:">0,<100"`
						Int64 int64 `map:"int64" validate:">0,<100"`

						Uint   uint   `map:"uint" validate:">0,<100"`
						Uint8  uint8  `map:"uint8" validate:">0,<100"`
						Uint16 uint16 `map:"uint16" validate:">0,<100"`
						Uint32 uint32 `map:"uint32" validate:">0,<100"`
						Uint64 uint64 `map:"uint64" validate:">0,<100"`

						Float32 float32 `map:"float32" validate:">0.5,<100.5"`
						Float64 float64 `map:"float64" validate:">0.5,<100.5"`
					}{},
					sourceMap: map[string]LazyDecoder{
						"int":   testDecoder(50),
						"int8":  testDecoder(50),
						"int16": testDecoder(50),
						"int32": testDecoder(50),
						"int64": testDecoder(50),

						"uint":   testDecoder(50),
						"uint8":  testDecoder(50),
						"uint16": testDecoder(50),
						"uint32": testDecoder(50),
						"uint64": testDecoder(50),

						"float32": testDecoder(50.5),
						"float64": testDecoder(50.5),
					},
				},
				{
					desc: "min/max should correctly return error for different types of numbers",
					structPtr: &struct {
						IntBelowMin   int   `map:"intBelowMin" validate:">100"`
						Int8BelowMin  int8  `map:"int8BelowMin" validate:">100"`
						Int16BelowMin int16 `map:"int16BelowMin" validate:">100"`
						Int32BelowMin int32 `map:"int32BelowMin" validate:">100"`
						Int64BelowMin int64 `map:"int64BelowMin" validate:">100"`

						IntAboveMax   int   `map:"intAboveMax" validate:"<10"`
						Int8AboveMax  int8  `map:"int8AboveMax" validate:"<10"`
						Int16AboveMax int16 `map:"int16AboveMax" validate:"<10"`
						Int32AboveMax int32 `map:"int32AboveMax" validate:"<10"`
						Int64AboveMax int64 `map:"int64AboveMax" validate:"<10"`

						UintBelowMin   uint   `map:"uintBelowMin" validate:">100"`
						Uint8BelowMin  uint8  `map:"uint8BelowMin" validate:">100"`
						Uint16BelowMin uint16 `map:"uint16BelowMin" validate:">100"`
						Uint32BelowMin uint32 `map:"uint32BelowMin" validate:">100"`
						Uint64BelowMin uint64 `map:"uint64BelowMin" validate:">100"`

						UintAboveMax   uint   `map:"uintAboveMax" validate:"<10"`
						Uint8AboveMax  uint8  `map:"uint8AboveMax" validate:"<10"`
						Uint16AboveMax uint16 `map:"uint16AboveMax" validate:"<10"`
						Uint32AboveMax uint32 `map:"uint32AboveMax" validate:"<10"`
						Uint64AboveMax uint64 `map:"uint64AboveMax" validate:"<10"`

						Float32BelowMin float32 `map:"float32BelowMin" validate:">100.5"`
						Float64BelowMin float64 `map:"float64BelowMin" validate:">100.5"`

						Float32AboveMax float32 `map:"float32AboveMax" validate:"<10.5"`
						Float64AboveMax float64 `map:"float64AboveMax" validate:"<10.5"`
					}{},
					sourceMap: map[string]LazyDecoder{
						"intBelowMin":   testDecoder(50),
						"int8BelowMin":  testDecoder(50),
						"int16BelowMin": testDecoder(50),
						"int32BelowMin": testDecoder(50),
						"int64BelowMin": testDecoder(50),

						"intAboveMax":   testDecoder(50),
						"int8AboveMax":  testDecoder(50),
						"int16AboveMax": testDecoder(50),
						"int32AboveMax": testDecoder(50),
						"int64AboveMax": testDecoder(50),

						"uintBelowMin":   testDecoder(50),
						"uint8BelowMin":  testDecoder(50),
						"uint16BelowMin": testDecoder(50),
						"uint32BelowMin": testDecoder(50),
						"uint64BelowMin": testDecoder(50),

						"uintAboveMax":   testDecoder(50),
						"uint8AboveMax":  testDecoder(50),
						"uint16AboveMax": testDecoder(50),
						"uint32AboveMax": testDecoder(50),
						"uint64AboveMax": testDecoder(50),

						"float32BelowMin": testDecoder(50.5),
						"float64BelowMin": testDecoder(50.5),

						"float32AboveMax": testDecoder(50.5),
						"float64AboveMax": testDecoder(50.5),
					},
					expectErrToContain: []string{
						"IntBelowMin",
						"Int8BelowMin",
						"Int16BelowMin",
						"Int32BelowMin",
						"Int64BelowMin",

						"IntAboveMax",
						"Int8AboveMax",
						"Int16AboveMax",
						"Int32AboveMax",
						"Int64AboveMax",

						"UintBelowMin",
						"Uint8BelowMin",
						"Uint16BelowMin",
						"Uint32BelowMin",
						"Uint64BelowMin",

						"UintAboveMax",
						"Uint8AboveMax",
						"Uint16AboveMax",
						"Uint32AboveMax",
						"Uint64AboveMax",

						"Float32BelowMin",
						"Float64BelowMin",
						"Float32AboveMax",
						"Float64AboveMax",
					},
				},
				{
					desc: "no error for max length on string",
					structPtr: &struct {
						Str string `map:"str" validate:"len<10"`
					}{},
					sourceMap: map[string]LazyDecoder{"str": testDecoder("foo")},
				},
				{
					desc: "error for max length on string",
					structPtr: &struct {
						Str string `map:"str" validate:"len<1"`
					}{},
					sourceMap:          map[string]LazyDecoder{"str": testDecoder("foo")},
					expectErrToContain: []string{"Str", "3", "<", "1"},
				},
				{
					desc: "no error for min length on string",
					structPtr: &struct {
						Str string `map:"str" validate:"len>2"`
					}{},
					sourceMap: map[string]LazyDecoder{"str": testDecoder("foo")},
				},
				{
					desc: "error for min length on string",
					structPtr: &struct {
						Str string `map:"str" validate:"len>4"`
					}{},
					sourceMap:          map[string]LazyDecoder{"str": testDecoder("foo")},
					expectErrToContain: []string{"Str", "3", ">", "4"},
				},
				{
					desc: "no error for length equal validation on string",
					structPtr: &struct {
						Str string `map:"str" validate:"len=3"`
					}{},
					sourceMap: map[string]LazyDecoder{"str": testDecoder("foo")},
				},
				{
					desc: "error for length equal validation on string",
					structPtr: &struct {
						Str string `map:"str" validate:"len=4"`
					}{},
					sourceMap:          map[string]LazyDecoder{"str": testDecoder("foo")},
					expectErrToContain: []string{"Str", "3", "=", "4"},
				},
				{
					desc: "no error for max length on slice",
					structPtr: &struct {
						Slice []int `map:"slice" validate:"len<10"`
					}{},
					sourceMap: map[string]LazyDecoder{"slice": testDecoder([]int{1, 2, 3})},
				},
				{
					desc: "error for max length on slice",
					structPtr: &struct {
						Slice []int `map:"slice" validate:"len<1"`
					}{},
					sourceMap:          map[string]LazyDecoder{"slice": testDecoder([]int{1, 2, 3})},
					expectErrToContain: []string{"Slice", "3", "<", "1"},
				},
				{
					desc: "no error for min length on slice",
					structPtr: &struct {
						Slice []int `map:"slice" validate:"len>2"`
					}{},
					sourceMap: map[string]LazyDecoder{"slice": testDecoder([]int{1, 2, 3})},
				},
				{
					desc: "error for min length on slice",
					structPtr: &struct {
						Slice []int `map:"slice" validate:"len>4"`
					}{},
					sourceMap:          map[string]LazyDecoder{"slice": testDecoder([]int{1, 2, 3})},
					expectErrToContain: []string{"Slice", "3", ">", "4"},
				},
				{
					desc: "no error for length equal validation on slice",
					structPtr: &struct {
						Slice []int `map:"slice" validate:"len=3"`
					}{},
					sourceMap: map[string]LazyDecoder{"slice": testDecoder([]int{1, 2, 3})},
				},
				{
					desc: "error for length equal validation on slice",
					structPtr: &struct {
						Slice []int `map:"slice" validate:"len=4"`
					}{},
					sourceMap:          map[string]LazyDecoder{"slice": testDecoder([]int{1, 2, 3})},
					expectErrToContain: []string{"Slice", "3", "=", "4"},
				},
				{
					desc: "no error for max length on map",
					structPtr: &struct {
						Map map[string]any `map:"map" validate:"len<10"`
					}{},
					sourceMap: map[string]LazyDecoder{"map": testDecoder(map[string]any{"a": 1, "b": 2, "c": 3})},
				},
				{
					desc: "error for max length on map",
					structPtr: &struct {
						Map map[string]any `map:"map" validate:"len<1"`
					}{},
					sourceMap:          map[string]LazyDecoder{"map": testDecoder(map[string]any{"a": 1, "b": 2, "c": 3})},
					expectErrToContain: []string{"Map", "3", "<", "1"},
				},
				{
					desc: "no error for min length on map",
					structPtr: &struct {
						Map map[string]any `map:"map" validate:"len>2"`
					}{},
					sourceMap: map[string]LazyDecoder{"map": testDecoder(map[string]any{"a": 1, "b": 2, "c": 3})},
				},
				{
					desc: "error for min length on map",
					structPtr: &struct {
						Map map[string]any `map:"map" validate:"len>4"`
					}{},
					sourceMap:          map[string]LazyDecoder{"map": testDecoder(map[string]any{"a": 1, "b": 2, "c": 3})},
					expectErrToContain: []string{"Map", "3", ">", "4"},
				},
				{
					desc: "no error for length equal validation on map",
					structPtr: &struct {
						Map map[string]any `map:"map" validate:"len=3"`
					}{},
					sourceMap: map[string]LazyDecoder{"map": testDecoder(map[string]any{"a": 1, "b": 2, "c": 3})},
				},
				{
					desc: "error for length equal validation map",
					structPtr: &struct {
						Map map[string]any `map:"map" validate:"len=4"`
					}{},
					sourceMap:          map[string]LazyDecoder{"map": testDecoder(map[string]any{"a": 1, "b": 2, "c": 3})},
					expectErrToContain: []string{"Map", "3", "=", "4"},
				},
			}

			for _, test := range tests {
				t.Run(test.desc, func(t *testing.T) {
					err := parseFromMap("map", test.structPtr, test.sourceMap)
					if test.expectErrToContain != nil {
						tt.AssertErrContains(t, err, test.expectErrToContain...)
						return
					}
					tt.AssertNoErr(t, err)
				})
			}
		})
	})

	t.Run("using the default tag", func(t *testing.T) {
		t.Run("should work for multiple types of fields", func(t *testing.T) {
			var user struct {
				ID       int    `map:"id"`
				Username string `map:"username" default:"defaultUsername"`
				Address  struct {
					Street  string `map:"street" default:"defaultStreet"`
					City    string `map:"city"`
					Country string `map:"country"`
				} `map:"address"`

				OptionalStruct struct {
					ID int `map:"id" default:"41"`
				} `map:"optional_struct"`
			}

			// These all these should be overwritten by the parser:
			user.ID = 43
			user.Address.Country = "presetCountry"
			user.OptionalStruct.ID = 42
			user.Username = "presetUsername"
			user.Address.Street = "presetStreet"

			err := parseFromMap("map", &user, map[string]LazyDecoder{
				"id": testDecoder(44),
				"address": testDecoder(map[string]any{
					"city":    "fakeCity",
					"country": "fakeCountry",
				}),
			})
			tt.AssertNoErr(t, err)

			tt.AssertEqual(t, user.ID, 44)
			tt.AssertEqual(t, user.Username, "defaultUsername")
			tt.AssertEqual(t, user.Address.Street, "defaultStreet")
			tt.AssertEqual(t, user.Address.City, "fakeCity")
			tt.AssertEqual(t, user.Address.Country, "fakeCountry")
			tt.AssertEqual(t, user.OptionalStruct.ID, 41)
		})
	})
}

// This test helper just generates a LazyDecoder from
// any input data that can be marshaled as JSON, for making
// it easier to describe the test cases.
func testDecoder(value any) LazyDecoder {
	return func(target any) error {
		bytes, err := json.Marshal(value)
		if err != nil {
			return err
		}
		return json.Unmarshal(bytes, target)
	}
}
