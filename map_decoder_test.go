package kparse

import (
	"encoding/json"
	"testing"

	tt "github.com/teamcollab-net/kparse/internal/testtools"
)

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

func TestMapTagDecoder(t *testing.T) {
	t.Run("should work for valid structs", func(t *testing.T) {
		var user struct {
			ID       int    `map:"id"`
			Username string `map:"username"`
			Address  struct {
				Street  string `map:"street"`
				City    string `map:"city"`
				Country string `map:"country"`
			} `map:"address"`
		}
		err := parseFromMap("map", &user, map[string]LazyDecoder{
			"id":       testDecoder(42),
			"username": testDecoder("fakeUsername"),
			"address": testDecoder(map[string]any{
				"street":  "fakeStreet",
				"city":    "fakeCity",
				"country": "fakeCountry",
			}),
		})
		tt.AssertNoErr(t, err)

		tt.AssertEqual(t, user.ID, 42)
		tt.AssertEqual(t, user.Username, "fakeUsername")
		tt.AssertEqual(t, user.Address.Street, "fakeStreet")
		tt.AssertEqual(t, user.Address.City, "fakeCity")
		tt.AssertEqual(t, user.Address.Country, "fakeCountry")
	})

	t.Run("should return error if we try to save something that is not a map into a nested struct", func(t *testing.T) {
		var user struct {
			ID       int    `map:"id"`
			Username string `map:"username"`
			Address  struct {
				Street  string `map:"street"`
				City    string `map:"city"`
				Country string `map:"country"`
			} `map:"address"`
		}
		err := parseFromMap("map", &user, map[string]LazyDecoder{
			"id":       testDecoder(42),
			"username": testDecoder("fakeUsername"),
			"address":  testDecoder("notAMap"),
		})

		tt.AssertErrContains(t, err, "string", "Address", "Street", "City", "Country")
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

		t.Run("validate min and max", func(t *testing.T) {
			tests := []struct {
				desc               string
				structPtr          any
				sourceMap          map[string]LazyDecoder
				expectErrToContain []string
			}{
				{
					desc: "should allow values in range",
					structPtr: &struct {
						Salary     int `map:"salary" validate:"min=1000"`
						AgeInYears int `map:"age" validate:"max=100"`
						HeightInCm int `map:"height" validate:"min=140,max=230"`
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
						ValidBefore int `map:"before" validate:"max=100"`
						BelowMin    int `map:"belowMin" validate:"min=1000"`
						ValidAfter  int `map:"after" validate:"min=140,max=230"`
					}{},
					sourceMap: map[string]LazyDecoder{
						"before":   testDecoder(45),
						"belowMin": testDecoder(500),
						"after":    testDecoder(178),
					},
					expectErrToContain: []string{"BelowMin", "min", "500", "1000"},
				},
				{
					desc: "should block if value above maximum",
					structPtr: &struct {
						ValidBefore int `map:"before" validate:"max=100"`
						AboveMax    int `map:"aboveMax" validate:"max=100"`
						ValidAfter  int `map:"after" validate:"min=140,max=230"`
					}{},
					sourceMap: map[string]LazyDecoder{
						"before":   testDecoder(45),
						"aboveMax": testDecoder(500),
						"after":    testDecoder(178),
					},
					expectErrToContain: []string{"AboveMax", "max", "500", "100"},
				},
				{
					desc: "should not fail the validation if value is missing",
					structPtr: &struct {
						ValidBefore  int `map:"before" validate:"max=100"`
						MissingValue int `map:"missing" validate:"max=100"`
						ValidAfter   int `map:"after" validate:"min=140,max=230"`
					}{},
					sourceMap: map[string]LazyDecoder{
						"before": testDecoder(45),
						"after":  testDecoder(178),
					},
				},
				{
					desc: "should not conflict with the required validation when field when all is valid",
					structPtr: &struct {
						ValidBefore int `map:"before" validate:"max=100"`
						Required    int `map:"required" validate:"required,max=100"`
						ValidAfter  int `map:"after" validate:"min=140,max=230"`
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
						ValidBefore int `map:"before" validate:"max=100"`
						Required    int `map:"required" validate:"required,max=100"`
						ValidAfter  int `map:"after" validate:"min=140,max=230"`
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
						ValidBefore int `map:"before" validate:"max=100"`
						Required    int `map:"required" validate:"required,max=100"`
						ValidAfter  int `map:"after" validate:"min=140,max=230"`
					}{},
					sourceMap: map[string]LazyDecoder{
						"before":   testDecoder(45),
						"required": testDecoder(120), // not in range
						"after":    testDecoder(178),
					},
					expectErrToContain: []string{"Required", "above", "max"},
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

	t.Run("parsing slices", func(t *testing.T) {
		tests := []struct {
			desc          string
			inputSlice    any
			expectedSlice any
		}{
			{
				desc: "should work for string slices",
				inputSlice: []string{
					"fakeUser1",
					"fakeUser2",
				},
				expectedSlice: []string{
					"fakeUser1",
					"fakeUser2",
				},
			},
		}

		for _, test := range tests {
			t.Run(test.desc, func(t *testing.T) {
				var user struct {
					ID    int      `map:"id"`
					Slice []string `map:"slice"`
				}

				err := parseFromMap("map", &user, map[string]LazyDecoder{
					"id":    testDecoder(42),
					"slice": testDecoder(test.inputSlice),
				})
				tt.AssertNoErr(t, err)

				tt.AssertEqual(t, user.ID, 42)
				tt.AssertEqual(t, user.Slice, test.expectedSlice)
			})
		}
	})
}
