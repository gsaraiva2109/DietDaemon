# Changelog

## [0.2.0-alpha.3](https://github.com/gsaraiva2109/DietDaemon/compare/v0.2.0-alpha.2...v0.2.0-alpha.3) (2026-07-18)


### Fixes

* **food:** guard catalog writes against implausible macros, add repair tool ([3efe4c1](https://github.com/gsaraiva2109/DietDaemon/commit/3efe4c1ca18c0e5a2e8b39689d2336fc12a292e2))
* **food:** guard catalog writes against implausible macros, add repair tool ([f5c23c4](https://github.com/gsaraiva2109/DietDaemon/commit/f5c23c4ff28cd3dd79f2212b29bb4f5199fc4d7a))
* **taco:** parse the official TACO/NEPA spreadsheet layout instead of rejecting it ([2e33d24](https://github.com/gsaraiva2109/DietDaemon/commit/2e33d244ad1f35e496d49d52e92431ba73de8d2f))
* **taco:** reject TACO_DATA_PATH files with a mismatched column layout ([34f446f](https://github.com/gsaraiva2109/DietDaemon/commit/34f446f3298cb02294924b550d884ea19c091728))

## [0.2.0-alpha.2](https://github.com/gsaraiva2109/DietDaemon/compare/v0.2.0-alpha.1...v0.2.0-alpha.2) (2026-07-18)


### Features

* **log:** add structured food picker as precise alternative to text parser ([305baa0](https://github.com/gsaraiva2109/DietDaemon/commit/305baa07041d1a8bfacf5dc3fcd3c3b4a97c3216))
* **log:** structured food picker, onboarding weight logging, bot emoji cleanup ([d4dccba](https://github.com/gsaraiva2109/DietDaemon/commit/d4dccbacb39da3d6ea668150fc22b6c5ea8fa510))


### Fixes

* fixed docker compose dietdaemon version tag ([b3d477d](https://github.com/gsaraiva2109/DietDaemon/commit/b3d477d40b02c08275c28e3cd1760a986436bb99))
* **onboarding:** log initial weight to weight_log on first-time completion ([1e9d169](https://github.com/gsaraiva2109/DietDaemon/commit/1e9d16923fd4a2976781cd68bc67181c3e18547b))
* **release:** continue alpha prerelease sequence ([7e2814b](https://github.com/gsaraiva2109/DietDaemon/commit/7e2814b32b0ae0b5192598a51a6e32b6da718836))

## [0.2.0-alpha.3](https://github.com/gsaraiva2109/DietDaemon/compare/v0.1.0-alpha.3...v0.2.0-alpha.3) (2026-07-18)


### Features

* add restore path for backups (issue [#95](https://github.com/gsaraiva2109/DietDaemon/issues/95)) ([56ede68](https://github.com/gsaraiva2109/DietDaemon/commit/56ede68b4527cdcefa658608265b3adc6727dd2f))
* **api:** add account data export and deletion endpoints ([ecdb353](https://github.com/gsaraiva2109/DietDaemon/commit/ecdb353cf1a44801bc335989cbfb537741eee077))
* **api:** add account data export and deletion endpoints ([ab4c476](https://github.com/gsaraiva2109/DietDaemon/commit/ab4c47672a4e04dffcd725eaa893aed038a49171)), closes [#96](https://github.com/gsaraiva2109/DietDaemon/issues/96)
* **auth:** gate registration on MULTI_USER flag ([17494c6](https://github.com/gsaraiva2109/DietDaemon/commit/17494c6cb46175ff11ea8106088230c878852617))
* **auth:** gate registration on MULTI_USER flag ([69cebb6](https://github.com/gsaraiva2109/DietDaemon/commit/69cebb6a36f3454e585166aa815fdb58ffb06db5)), closes [#98](https://github.com/gsaraiva2109/DietDaemon/issues/98)
* **backup:** add List/Read to localdisk and s3dest destinations ([39e39f0](https://github.com/gsaraiva2109/DietDaemon/commit/39e39f047ee4aa5a8be64749f51de0e467dacd7e))
* **backup:** export weight/measurements/sleep/workouts/water/fasting/photos ([e91f8e9](https://github.com/gsaraiva2109/DietDaemon/commit/e91f8e9c4abccb1a6d0cff8ec0be65c2058f3869))
* **exportfmt:** add CSV writers/readers for all 9 trackable entities ([8c75471](https://github.com/gsaraiva2109/DietDaemon/commit/8c75471e58f6da80d0dafe6e82b2388c9f630e5c))
* **foods:** add private custom foods ([96e267e](https://github.com/gsaraiva2109/DietDaemon/commit/96e267e5187ee24ec123033018358c1d0dd2b125))
* **foods:** add private custom foods ([8ae8c7f](https://github.com/gsaraiva2109/DietDaemon/commit/8ae8c7fd434afa935500742c4982387da43830c7))
* **foods:** filter custom foods ([0bcf919](https://github.com/gsaraiva2109/DietDaemon/commit/0bcf9199419d8691bdcfd2ff457aa3014e25c230))
* **ocr:** OCR-assisted nutrition-label capture ([bd98e38](https://github.com/gsaraiva2109/DietDaemon/commit/bd98e38758db99dd36ed759f90d3384f1b924506))
* **ocr:** OCR-assisted nutrition-label capture backend ([#87](https://github.com/gsaraiva2109/DietDaemon/issues/87)) ([c4dde4c](https://github.com/gsaraiva2109/DietDaemon/commit/c4dde4c4137818eaa7abb3e08208e9ffcafbffdc))
* **ocr:** OCR-assisted nutrition-label capture UI ([#87](https://github.com/gsaraiva2109/DietDaemon/issues/87)) ([d0c73a8](https://github.com/gsaraiva2109/DietDaemon/commit/d0c73a81be7955427c3c7481f61c109df9cd97b9))
* **restore:** add cmd/restore CLI ([83af3db](https://github.com/gsaraiva2109/DietDaemon/commit/83af3db9c14f8b57f006294619e6fddf089d5fdd))
* **restore:** add internal/restore orchestrator package ([0c10dff](https://github.com/gsaraiva2109/DietDaemon/commit/0c10dff53b5b1f4d223da4becc1f24194b4ce6bb))
* **store:** add idempotent restore methods and range queries ([8f4d50b](https://github.com/gsaraiva2109/DietDaemon/commit/8f4d50bbaa1eed513c2887dda029336d76edacb8))


### Fixes

* **api:** log mailer send failures instead of discarding them ([7bde129](https://github.com/gsaraiva2109/DietDaemon/commit/7bde129144b4f9e29f16383cee1d009ec0baf99c))
* **api:** require current password to change account email ([90b0e69](https://github.com/gsaraiva2109/DietDaemon/commit/90b0e69bfe91bbb660967aa655e50419a52824f4))
* **api:** use constant-time comparison for OIDC state token ([1d1a31e](https://github.com/gsaraiva2109/DietDaemon/commit/1d1a31e5f8a59f2f744f9131d0c659dcc29a3885))
* **api:** use CSPRNG for handler ID generation ([fd752d0](https://github.com/gsaraiva2109/DietDaemon/commit/fd752d05e16a8fd3cc67782a97b131693d258a61))
* **api:** wire COOKIE_DOMAIN config through to session cookies ([d7b33dc](https://github.com/gsaraiva2109/DietDaemon/commit/d7b33dc9a5c488411594a11926bb52d28fbf9523))
* **auth:** close IP spoofing, TOTP brute-force, and timing-leak gaps ([dc1b2a8](https://github.com/gsaraiva2109/DietDaemon/commit/dc1b2a87657bc55b227e887d7ee8eae1ba600cad))
* **auth:** perform dummy KDF on malformed hash to avoid timing leak ([cbaa061](https://github.com/gsaraiva2109/DietDaemon/commit/cbaa061c09ddf4cbdec85260739e519aa8a34107))
* **deploy:** pin Compose image and add Postgres profile ([48c3d94](https://github.com/gsaraiva2109/DietDaemon/commit/48c3d94e7239fc23fe5a759e09849a75e3a47528))
* **deploy:** pin Compose image and add Postgres profile ([93b1f8a](https://github.com/gsaraiva2109/DietDaemon/commit/93b1f8a8b220d45e461eb322849319b77272001a))
* **foodimport:** omit zero failed backfill field ([609d0a0](https://github.com/gsaraiva2109/DietDaemon/commit/609d0a0d44d1bb0f86a4de0cd3bfaf35092b1171))
* **foods:** reset library source filter ([9eb4336](https://github.com/gsaraiva2109/DietDaemon/commit/9eb4336e28d5da0114f36686b3200fee4baad86b))
* **scheduler,store,config:** resolve rule bypass, rollback error, WebAuthn validation, stale doc ([77e0c9e](https://github.com/gsaraiva2109/DietDaemon/commit/77e0c9e729abdd32bf49afddb6eac11848963ab3))
* security and correctness issues (auth, timing, dialect) ([a3fabc0](https://github.com/gsaraiva2109/DietDaemon/commit/a3fabc0a673ec5bb6a20acc87dd9ba0badc718b6))
* **store:** use dialect-aware date truncation instead of raw SQLite date() ([a1e33d1](https://github.com/gsaraiva2109/DietDaemon/commit/a1e33d1e6f093fded51531fa5cf12f010d929ee3))
* **web:** recover from stale Vite chunks ([cd791b2](https://github.com/gsaraiva2109/DietDaemon/commit/cd791b2d26bfafb7db0d86ed01465f82e6912a25))
* **web:** recover from stale Vite chunks ([d021d60](https://github.com/gsaraiva2109/DietDaemon/commit/d021d6093b0d238a00c04c25b9c6ea0c5a60571f))
* **web:** reload on chunk import failure ([ee65ad8](https://github.com/gsaraiva2109/DietDaemon/commit/ee65ad8b6feea184b1ae13866747f7e5cd4aa220))
* **web:** reload on chunk import failure ([f1145a5](https://github.com/gsaraiva2109/DietDaemon/commit/f1145a562ff779e5322c26648d53e524e089d8dc))


### Performance

* batch processing and defer dashboard charts ([aca6663](https://github.com/gsaraiva2109/DietDaemon/commit/aca66630b416570233a999d6a5a345bc7834095d))
* optimize backend processing ([985463d](https://github.com/gsaraiva2109/DietDaemon/commit/985463dcd696910f0fa3f2e92dcd356282e3481f))
* **web:** defer dashboard chart code ([880ab3b](https://github.com/gsaraiva2109/DietDaemon/commit/880ab3b502d58a374e146396a983017884c7c799))

## [v0.1.0-alpha.3] - 2026-07-15

### Features

- Automatically pull the required Ollama chat and embedding models at startup.

## [v0.1.0-alpha.2] - 2026-07-14

### Features

- Search the imported food catalog and save foods to a personal library.
- Add bulk-import status, MyFitnessPal log import, ingredient-aware suggestions, and shareable links.

### Fixes

- Improve assistant session titles, food matching, and error messages.
- Show individual food-embedding backfill failures for easier recovery.

## [v0.1.0-alpha.1] - 2026-07-13

### Features

- Add an AI meal assistant with chat, food logging, and OpenAI, Anthropic, and Ollama providers.
- Add the embedded dashboard, English and Brazilian Portuguese support, smart reminders, and correction feedback.
- Add bulk food import with unchanged-file skipping.

### Fixes

- Improve authentication routing and chat-session reliability.

### Performance

- Load dashboard routes on demand and self-host fonts for faster page loads.

### Security

- Scope AI chat sessions to their owner and harden authentication handling.
