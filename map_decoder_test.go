package configparser

import (
	"testing"

	tt "github.com/blackpointcyber/configparser/internal/testtools"
	"github.com/vingarcia/structscanner"
)

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
		err := structscanner.Decode(&user, newMapTagDecoder("map", map[any]any{
			"id":       42,
			"username": "fakeUsername",
			"address": map[any]any{
				"street":  "fakeStreet",
				"city":    "fakeCity",
				"country": "fakeCountry",
			},
		}))
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
		err := structscanner.Decode(&user, newMapTagDecoder("map", map[any]any{
			"id":       42,
			"username": "fakeUsername",
			"address":  "notAMap",
		}))

		tt.AssertErrContains(t, err, "string", "Address", "Street", "City", "Country")
	})

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

			err := structscanner.Decode(&user, newMapTagDecoder("map", map[any]any{
				"id": 44,
				"address": map[any]any{
					"city":    "fakeCity",
					"country": "fakeCountry",
				},
			}))
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
				input              map[any]any
				expectErrToContain []string
			}{
				{
					desc: "required field missing on root map",
					input: map[any]any{
						"id": 42,
						"address": map[any]any{
							"street":  "fakeStreet",
							"city":    "fakeCity",
							"country": "fakeCountry",
						},
					},
					expectErrToContain: []string{"missing", "required", "config", "username"},
				},
				{
					desc: "required field missing on nested map",
					input: map[any]any{
						"id":       42,
						"username": "fakeUsername",
						"address": map[any]any{
							"city":    "fakeCity",
							"country": "fakeCountry",
						},
					},
					expectErrToContain: []string{"missing", "required", "config", "street"},
				},
				{
					desc: "required field missing is a map",
					input: map[any]any{
						"id":       42,
						"username": "fakeUsername",
					},
					expectErrToContain: []string{"missing", "required", "config", "address"},
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
					err := structscanner.Decode(&user, newMapTagDecoder("map", test.input))

					tt.AssertErrContains(t, err, test.expectErrToContain...)
				})
			}
		})

		t.Run("should return error if the validation is misspelled", func(t *testing.T) {
			var user struct {
				ID       int    `map:"id"`
				Username string `map:"username" validate:"not_required"`
			}
			err := structscanner.Decode(&user, newMapTagDecoder("map", map[any]any{
				"id":       42,
				"username": "fakeUsername",
			}))

			tt.AssertErrContains(t, err, "validation", "not_required")
		})

		t.Run("should not return error if the required field is empty but has a default value", func(t *testing.T) {
			var user struct {
				ID       int    `map:"id"`
				Username string `map:"username" validate:"required" default:"defaultUsername"`
			}
			err := structscanner.Decode(&user, newMapTagDecoder("map", map[any]any{
				"id": 42,
			}))
			tt.AssertNoErr(t, err)

			tt.AssertEqual(t, user.ID, 42)
			tt.AssertEqual(t, user.Username, "defaultUsername")
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

			err := structscanner.Decode(&user, newMapTagDecoder("map", map[any]any{
				"id": 44,
				"address": map[any]any{
					"city":    "fakeCity",
					"country": "fakeCountry",
				},
			}))
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

				err := structscanner.Decode(&user, newMapTagDecoder("map", map[any]any{
					"id":    42,
					"slice": test.inputSlice,
				}))
				tt.AssertNoErr(t, err)

				tt.AssertEqual(t, user.ID, 42)
				tt.AssertEqual(t, user.Slice, test.expectedSlice)
			})
		}
	})
}
