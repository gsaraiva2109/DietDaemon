# Dashboard prototype — verdict

Question: what structure should the "Today" screen use?
Explored 3 variants via ?variant=A|B|C (prototype skill, UI branch).

**Chosen: Variant A — Ring-focused.** Apple-Health-style hero calories ring
(remaining-to-target) + 4 satellite macro rings + meals timeline. Best matches
the locked reference (Apple Health) and the hero job ("remaining macros at a
glance"). Folded into src/routes/Dashboard.tsx; variants B (editorial split)
and C (stacked bars) and the switcher were deleted.

Throwaway dev helpers kept temporarily for screenshotting other screens:
dev-mock-api.mjs, shoot.mjs, shots/ — remove before shipping (Task 5 cleanup).
