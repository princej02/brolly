# 🎟️ Ticket: Brolly – "Do I Need an Umbrella?" CLI (ID: ENG-3263)

**Difficulty Level:** Level 3 (Practical APIs) | **Language:** Go (1.23+, standard library)

---

## 💡 The Concept Explained

**Consuming an external API** means your program acts as the *client*: it builds a URL, sends an HTTP request to someone else's server, waits a bounded amount of time, then turns their JSON into your own types. The key habit is that the outside world's data shape stops at the edge of your app. Your infrastructure layer translates "their JSON" into "your structs", so your core logic never knows (or cares) that Open-Meteo exists.

- Go `net/http` client docs: https://pkg.go.dev/net/http#Client
- Open-Meteo Forecast API: https://open-meteo.com/en/docs
- Open-Meteo Geocoding API: https://open-meteo.com/en/docs/geocoding-api

No API key needed for either endpoint.

---

## 📋 Context & Goal

Northern England weather has one rule: it will change its mind. You keep leaving the house without a jacket at 3pm and regretting it at 7pm.

Build **Brolly**, a tiny CLI that takes a city name, looks up the forecast for the next N hours, and gives you a straight verdict: umbrella or not, jacket or not, and *why*.

The real lesson: two chained API calls, a response format that is awkward on purpose (parallel arrays), and a core decision engine that you can test without ever touching the network.

---

## 🔍 How It Works & Expected Output

### Input / Flow

```text
 $ brolly --city Liverpool --hours 12
            │
            ▼
┌──────────────────────────────┐
│ TRANSPORT  cmd/brolly        │  parse + validate flags (hours 1..48)
└──────────────┬───────────────┘
               │ "Liverpool"
               ▼
┌──────────────────────────────┐
│ INFRA  internal/openmeteo    │  GET geocoding-api ... ?name=Liverpool&count=1
│   1. Geocode(city)           │  → Place{Name, Country, Lat, Lon, Timezone}
│   2. Hourly(place, hours)    │  GET api.open-meteo.com/v1/forecast ...
│                              │  parallel arrays → []forecast.Hour
└──────────────┬───────────────┘
               │ []forecast.Hour  (your type, not theirs)
               ▼
┌──────────────────────────────┐
│ CORE  internal/forecast      │  Advise(hours, rules) → Advice
│   pure, no net/http, no os   │  umbrella? jacket? reasons, peaks
└──────────────┬───────────────┘
               │ Advice
               ▼
┌──────────────────────────────┐
│ TRANSPORT  cmd/brolly        │  render text, choose exit code
└──────────────────────────────┘
```

### The two requests (for reference)

```text
GET https://geocoding-api.open-meteo.com/v1/search?name=Liverpool&count=1

GET https://api.open-meteo.com/v1/forecast
    ?latitude=53.41&longitude=-2.98
    &hourly=temperature_2m,precipitation_probability,wind_speed_10m
    &forecast_hours=12
    &timezone=auto
```

Heads up: the forecast comes back as **columns**, not rows:

```json
"hourly": {
  "time":                      ["2026-09-26T21:00", "2026-09-26T22:00", "..."],
  "temperature_2m":            [13.1, 12.4, "..."],
  "precipitation_probability": [20, 35, "..."],
  "wind_speed_10m":            [18.2, 21.0, "..."]
}
```

Turning that into `[]Hour` rows is *infra's* job.

### Example Output / Preview

**Happy path:**

```text
$ brolly --city Liverpool --hours 12
📍 Liverpool, United Kingdom (53.41, -2.98)
🕒 Next 12 hours from Sat 21:00

   Rain chance peaks at 70% around 03:00
   Coldest 8.4°C · Windiest 34 km/h

☂️  Take an umbrella   (rain chance ≥ 50%)
🧥  Take a jacket      (below 12°C, wind ≥ 30 km/h)

$ echo $?
0
```

**Nice day:**

```text
$ brolly --city Seville --hours 6
📍 Seville, Spain (37.38, -5.97)
🕒 Next 6 hours from Sat 22:00

   Rain chance peaks at 0% around 22:00
   Coldest 19.2°C · Windiest 9 km/h

😎  Leave both at home.
```

**City not found:**

```text
$ brolly --city Atlantisss
brolly: no place found matching "Atlantisss"
$ echo $?
1
```

**Network / API trouble (e.g. wifi off):**

```text
$ brolly --city Leeds
brolly: forecast service unavailable: request timed out after 5s
$ echo $?
2
```

**Bad input:**

```text
$ brolly --city Leeds --hours 99
brolly: --hours must be between 1 and 48 (got 99)
$ echo $?
64
```

---

## 🧱 Separation of Concerns Blueprint

```text
brolly/
├── go.mod                          module github.com/princej02/brolly
├── README.md
├── cmd/
│   └── brolly/
│       └── main.go                 TRANSPORT: flags, wiring, printing, exit codes
└── internal/
    ├── forecast/                   CORE: pure rules, zero I/O
    │   ├── advice.go
    │   └── advice_test.go
    └── openmeteo/                  INFRA: HTTP + JSON translation
        ├── client.go
        ├── client_test.go
        └── testdata/
            ├── geocode_liverpool.json
            ├── geocode_empty.json
            ├── forecast_rainy.json
            └── forecast_mismatched.json
```

### Contracts to build against (shapes, not implementations)

```go
// internal/forecast (CORE)
type Hour struct {
    At            time.Time
    TempC         float64
    RainChancePct int
    WindKmh       float64
}

type Rules struct {
    UmbrellaAtPct int     // default 50
    JacketBelowC  float64 // default 12
    JacketWindKmh float64 // default 30
}

type Advice struct { /* you design this */ }

func Advise(hours []Hour, r Rules) (Advice, error)
```

```go
// internal/openmeteo (INFRA)
type Place struct { /* you design this */ }

func (c *Client) Geocode(ctx context.Context, city string) (Place, error)
func (c *Client) Hourly(ctx context.Context, p Place, hours int) ([]forecast.Hour, error)
```

**Dependency direction:** `openmeteo` imports `forecast`. `forecast` imports nothing of yours. If you ever feel the urge to import `net/http` inside `forecast/`, stop and rethink.

---

## ✅ Acceptance Criteria

1. `brolly --city <name> [--hours N]` works, `--hours` defaults to 12, valid range 1..48.
2. Two real HTTP calls, chained: geocode first, then forecast using the returned coordinates.
3. Every request has a timeout (5s total is fine) driven by `context`.
4. The column arrays are converted into `[]forecast.Hour` inside `openmeteo`. Mismatched array lengths are an **error**, not a panic and not silent truncation.
5. Times are parsed in the **place's** timezone, not your laptop's.
6. `forecast.Advise` is pure and fully table-tested: umbrella only, jacket from cold only, jacket from wind only, both, neither, empty input.
7. `openmeteo` is tested against an `httptest.Server` serving your `testdata/` fixtures. **No test hits the real internet.**
8. Distinct, typed-or-sentinel errors for "not found" vs "upstream failure" so `main.go` can pick exit codes `1` vs `2` using `errors.Is` / `errors.As`.
9. Exit code decisions live only in `cmd/brolly/main.go`.
10. `gofmt`, `go vet ./...` and `go test ./...` all clean. README shows one real sample run.

---

## 🧭 Nudges (not solutions)

- **Tests first, again.** Write `advice_test.go` before `advice.go`. What should `Advise` return for zero hours: an error, or an "all clear" `Advice`? Decide in the test, then make it pass.
- **The base URL is a dependency.** How will your test point the client at `httptest.Server` instead of open-meteo.com? Look at what your `Client` struct holds.
- **The zero-value trap.** `http.DefaultClient` has no timeout at all. Where does your timeout actually come from?
- **Status codes aren't errors in Go.** A 500 response returns `err == nil`. What do you check before decoding?
- **Empty geocode results.** When nothing matches, does the response contain `"results": []`, or is the key missing entirely? Save a real response as a fixture and find out rather than guessing.
- **Nulls in the columns.** Can `precipitation_probability` contain `null`? What would `[]int` do with that when decoding? What would `[]*int` do?
- **Timezone gotcha.** `"2026-09-26T21:00"` has no offset. Which `time` function parses a string as if it were in a given location? Where does that location come from?
- **Defaults vs rules.** Should `Rules` defaults live in `forecast` (as a `DefaultRules()`), or in `main.go`? Argue it either way in your README in one sentence.

---

## 🌶️ Stretch (only after sign-off criteria pass)

- `--json` flag that prints `Advice` as JSON (the core type shouldn't need changing for this; if it does, ask why).
- Pretty output with a third-party lib such as `github.com/charmbracelet/lipgloss`, kept strictly inside `cmd/`.
- Cache the geocode result for a city in a local JSON file so repeated runs only make one request.

---

## 📦 Definition of Done

- Repo pushed to `github.com/princej02/brolly`
- `go test ./...` green, and you've actually run it
- Reply **done** with the link, and I'll clone, run and review it