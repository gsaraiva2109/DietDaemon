# Changelog

## [0.2.0-alpha.7](https://github.com/gsaraiva2109/DietDaemon/compare/v0.2.0-alpha.6...v0.2.0-alpha.7) (2026-08-08)


### Fixes

* **ci:** set missing sha output in release Docker metadata step ([#286](https://github.com/gsaraiva2109/DietDaemon/issues/286)) ([31c4311](https://github.com/gsaraiva2109/DietDaemon/commit/31c4311697185e521555e5a49c3bcb7a21207cde))

## [0.2.0-alpha.6](https://github.com/gsaraiva2109/DietDaemon/compare/v0.2.0-alpha.5...v0.2.0-alpha.6) (2026-08-08)


### Features

* add photo menu dining mode ([#218](https://github.com/gsaraiva2109/DietDaemon/issues/218)) ([b1095ad](https://github.com/gsaraiva2109/DietDaemon/commit/b1095ad55da581f170d574900c71d09aff84f18a))
* **api:** add photo mime allowlist, quota, and view validation ([#214](https://github.com/gsaraiva2109/DietDaemon/issues/214)) ([6a107bb](https://github.com/gsaraiva2109/DietDaemon/commit/6a107bb427ecd8c3c9487a49b23e9af359d78418)), closes [#208](https://github.com/gsaraiva2109/DietDaemon/issues/208)
* **plan-import:** extract native PDF text before falling back to scan ([#221](https://github.com/gsaraiva2109/DietDaemon/issues/221)) ([#253](https://github.com/gsaraiva2109/DietDaemon/issues/253)) ([f02d398](https://github.com/gsaraiva2109/DietDaemon/commit/f02d398cbdb44f03eabfcd38f1639ffd7b89c0f3))
* **plan-import:** preserve weekday schedule and substitutions from source ([#254](https://github.com/gsaraiva2109/DietDaemon/issues/254)) ([892cddf](https://github.com/gsaraiva2109/DietDaemon/commit/892cddf797e805a5ca7ef5f6f7a1465da9838c1d))
* suggest target review from sustained weight-trend divergence ([#217](https://github.com/gsaraiva2109/DietDaemon/issues/217)) ([7c331c0](https://github.com/gsaraiva2109/DietDaemon/commit/7c331c0b6f932122985e7ef0a8edb69aad21385d))
* **vision:** support multi-page diet plan extraction ([#222](https://github.com/gsaraiva2109/DietDaemon/issues/222)) ([#252](https://github.com/gsaraiva2109/DietDaemon/issues/252)) ([105f0d0](https://github.com/gsaraiva2109/DietDaemon/commit/105f0d09eedc4038e89b09de70f89cfe174a31b9))


### Fixes

* allow inline styles and data: images in CSP ([#267](https://github.com/gsaraiva2109/DietDaemon/issues/267)) ([8b0fa28](https://github.com/gsaraiva2109/DietDaemon/commit/8b0fa2837ffcf9ec32aa209077de89f4f0a7e379))
* **assistant:** wire mailer into background purge runner ([#212](https://github.com/gsaraiva2109/DietDaemon/issues/212)) ([2b80c1d](https://github.com/gsaraiva2109/DietDaemon/commit/2b80c1d88cbc98f3402989c15a10ed86da1d6809))
* **auth:** stop lockout self-perpetuation and enforce account-deletion status at login ([#280](https://github.com/gsaraiva2109/DietDaemon/issues/280)) ([f0fbb60](https://github.com/gsaraiva2109/DietDaemon/commit/f0fbb60b683e18c9dd5ed80e334b6907be91548f))
* **deps:** replace dependency framer-motion with motion ([#235](https://github.com/gsaraiva2109/DietDaemon/issues/235)) ([09f48f6](https://github.com/gsaraiva2109/DietDaemon/commit/09f48f6d0020822cc4c1721365ef2bac1c051c41))
* **deps:** unblock compatible frontend updates ([#248](https://github.com/gsaraiva2109/DietDaemon/issues/248)) ([107db5b](https://github.com/gsaraiva2109/DietDaemon/commit/107db5b4e20c7d362603c43687b0211234495cde))
* **deps:** update dependency motion to v13 ([#241](https://github.com/gsaraiva2109/DietDaemon/issues/241)) ([73a5559](https://github.com/gsaraiva2109/DietDaemon/commit/73a5559791b471c90f0d67e6900ee18578a8a721))
* **deps:** update go dependencies ([#237](https://github.com/gsaraiva2109/DietDaemon/issues/237)) ([f1062bb](https://github.com/gsaraiva2109/DietDaemon/commit/f1062bbbc4fe05b90ddee1a5ab576cc24e3f9e5d))
* **deps:** update go dependencies ([#255](https://github.com/gsaraiva2109/DietDaemon/issues/255)) ([187a956](https://github.com/gsaraiva2109/DietDaemon/commit/187a9567693310792d5db05101999d289e8f3872))
* **deps:** update go dependencies ([#262](https://github.com/gsaraiva2109/DietDaemon/issues/262)) ([08152a9](https://github.com/gsaraiva2109/DietDaemon/commit/08152a9710a536f9e2c293d589578b95109a5339))
* **deps:** update go dependencies to v0.44.0 ([#265](https://github.com/gsaraiva2109/DietDaemon/issues/265)) ([61f385f](https://github.com/gsaraiva2109/DietDaemon/commit/61f385fb7dc154fb0407a6ccc1ad954464f29e0f))
* **deps:** update npm dependencies ([#238](https://github.com/gsaraiva2109/DietDaemon/issues/238)) ([37bde3c](https://github.com/gsaraiva2109/DietDaemon/commit/37bde3c9726c67bbe5f887c0e58b83fda65b4634))
* **goals:** simplify target review guard ([#251](https://github.com/gsaraiva2109/DietDaemon/issues/251)) ([1654fe1](https://github.com/gsaraiva2109/DietDaemon/commit/1654fe19347ba4a1034ed2e1bbfdb03e327d75e1))
* **mailer:** migrate Resend to v3 ([#250](https://github.com/gsaraiva2109/DietDaemon/issues/250)) ([f8afeb0](https://github.com/gsaraiva2109/DietDaemon/commit/f8afeb07bc0ac5edb95dfa8809cfc3da3562612b))
* **mailer:** surface send failures and add SMTP timeouts ([#278](https://github.com/gsaraiva2109/DietDaemon/issues/278)) ([f6390e8](https://github.com/gsaraiva2109/DietDaemon/commit/f6390e86c263fe4819151ca769921c7ad4d11e1a))
* per-user timezone in API reads and backup deletion on account purge ([#281](https://github.com/gsaraiva2109/DietDaemon/issues/281)) ([4945b08](https://github.com/gsaraiva2109/DietDaemon/commit/4945b086e1de06268fb386cc72f686126c5164b2))
* **store:** make meal save and rollup update atomic and additive ([#279](https://github.com/gsaraiva2109/DietDaemon/issues/279)) ([c572813](https://github.com/gsaraiva2109/DietDaemon/commit/c5728131741ed48e412f1e4fa0e8c0963eb2d15f)), closes [#272](https://github.com/gsaraiva2109/DietDaemon/issues/272)
* **web:** resolve SonarQube reliability issues, raise new-code coverage ([#198](https://github.com/gsaraiva2109/DietDaemon/issues/198)) ([5071222](https://github.com/gsaraiva2109/DietDaemon/commit/50712220ebdfc2fed8a8c9f035ff958bb99fe0e7))
* **web:** stop polling failed queries ([#206](https://github.com/gsaraiva2109/DietDaemon/issues/206)) ([c5764e8](https://github.com/gsaraiva2109/DietDaemon/commit/c5764e8cf48f5c07bdc6c8d15b6c4840628b2286))


### Security

* **web:** upgrade react router to v8 ([#246](https://github.com/gsaraiva2109/DietDaemon/issues/246)) ([e57c3ec](https://github.com/gsaraiva2109/DietDaemon/commit/e57c3ec0e8f7d327f6b7e04f3fd2716d38672b22))

## [0.2.0-alpha.5](https://github.com/gsaraiva2109/DietDaemon/compare/v0.2.0-alpha.4...v0.2.0-alpha.5) (2026-07-28)


### Features

* **plan:** diet plan import via photo/PDF extraction ([#194](https://github.com/gsaraiva2109/DietDaemon/issues/194)) ([#197](https://github.com/gsaraiva2109/DietDaemon/issues/197)) ([f169b04](https://github.com/gsaraiva2109/DietDaemon/commit/f169b046f84160222c5bd11d436e9f8275449a65))


### Performance

* **store:** fix N+1 queries, Postgres unique-violation bug, missing indexes ([#183](https://github.com/gsaraiva2109/DietDaemon/issues/183)) ([c065ef2](https://github.com/gsaraiva2109/DietDaemon/commit/c065ef2b921d1daecb2e3e7ae660509e4053c258))

## [0.2.0-alpha.4](https://github.com/gsaraiva2109/DietDaemon/compare/v0.2.0-alpha.3...v0.2.0-alpha.4) (2026-07-24)


### Features

* **api:** add admin endpoints to trigger food-import/repair/backfill ([#148](https://github.com/gsaraiva2109/DietDaemon/issues/148)) ([b8111c5](https://github.com/gsaraiva2109/DietDaemon/commit/b8111c5651aef6a96244db7dd4ab098dd3e94378))
* **config:** add LoadMinimal for CLI tools that don't need daemon validation ([#147](https://github.com/gsaraiva2109/DietDaemon/issues/147)) ([8209e8a](https://github.com/gsaraiva2109/DietDaemon/commit/8209e8a3d4bb4da0190fe267665c19909c9ca926))
* **config:** move operational constants into config system ([#145](https://github.com/gsaraiva2109/DietDaemon/issues/145)) ([01fcd9c](https://github.com/gsaraiva2109/DietDaemon/commit/01fcd9c694449507f3242a0fd62a18f0110e2f6b))
* **targets:** add configurable water_goal_ml to daily targets ([#146](https://github.com/gsaraiva2109/DietDaemon/issues/146)) ([8ec2c1f](https://github.com/gsaraiva2109/DietDaemon/commit/8ec2c1f1ba4714c32d925bfd25bfaefd185b0145))


### Fixes

* **http:** harden API and dashboard boundaries ([#141](https://github.com/gsaraiva2109/DietDaemon/issues/141)) ([ea78fe7](https://github.com/gsaraiva2109/DietDaemon/commit/ea78fe779af0c3b0612d8bd995f1068c86ac93eb))
* **meals:** resolve template prefix leak, missing macro totals, share PNG corners ([#153](https://github.com/gsaraiva2109/DietDaemon/issues/153)) ([67b0b2b](https://github.com/gsaraiva2109/DietDaemon/commit/67b0b2b010e934be086215bd45dca66d7da903db)), closes [#137](https://github.com/gsaraiva2109/DietDaemon/issues/137)
* **sonarqube:** match projectKey to renamed SonarQube project key ([#157](https://github.com/gsaraiva2109/DietDaemon/issues/157)) ([ea4ba7d](https://github.com/gsaraiva2109/DietDaemon/commit/ea4ba7d55f0bc3258b31a0d4b58061c60abdb745))
* **store:** compute meal and water rollup dates in user's local timezone ([#151](https://github.com/gsaraiva2109/DietDaemon/issues/151)) ([6fed99a](https://github.com/gsaraiva2109/DietDaemon/commit/6fed99a42f295e9c78ef6fdbf0cc135dcf81f924)), closes [#143](https://github.com/gsaraiva2109/DietDaemon/issues/143)


### Security

* **http:** harden server and auth failures ([#139](https://github.com/gsaraiva2109/DietDaemon/issues/139)) ([b3f5fbf](https://github.com/gsaraiva2109/DietDaemon/commit/b3f5fbfb16dd2c1c1a1d682076c29fd96b839166))

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
