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
		err := structscanner.Decode(&user, newMapTagDecoder("map", map[string]interface{}{
			"id":       42,
			"username": "fakeUsername",
			"address": map[string]interface{}{
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
		err := structscanner.Decode(&user, newMapTagDecoder("map", map[string]interface{}{
			"id":       42,
			"username": "fakeUsername",
			"address":  "notAMap",
		}))

		tt.AssertErrContains(t, err, "string", "Address", "Street", "City", "Country")
	})

	t.Run("using the require modifier", func(t *testing.T) {
		t.Run("should ignore missing fields if they are not required", func(t *testing.T) {
			var user struct {
				ID       int    `map:"id"`
				Username string `map:"username"`
				Address  struct {
					Street  string `map:"street"`
					City    string `map:"city"`
					Country string `map:"country"`
				} `map:"address"`
			}
			err := structscanner.Decode(&user, newMapTagDecoder("map", map[string]interface{}{
				"id": 42,
				"address": map[string]interface{}{
					"city":    "fakeCity",
					"country": "fakeCountry",
				},
			}))
			tt.AssertNoErr(t, err)

			tt.AssertEqual(t, user.ID, 42)
			tt.AssertEqual(t, user.Username, "")
			tt.AssertEqual(t, user.Address.Street, "")
			tt.AssertEqual(t, user.Address.City, "fakeCity")
			tt.AssertEqual(t, user.Address.Country, "fakeCountry")
		})

		t.Run("should return error for missing fields if they are required", func(t *testing.T) {
			var user struct {
				ID       int    `map:"id"`
				Username string `map:"username,required"`
				Address  struct {
					Street  string `map:"street"`
					City    string `map:"city"`
					Country string `map:"country"`
				} `map:"address"`
			}
			err := structscanner.Decode(&user, newMapTagDecoder("map", map[string]interface{}{
				"id": 42,
				"address": map[string]interface{}{
					"city":    "fakeCity",
					"country": "fakeCountry",
				},
			}))

			tt.AssertErrContains(t, err, "missing", "required", "username")
		})
	})
}
