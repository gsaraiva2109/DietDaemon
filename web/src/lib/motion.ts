// Motion presets. Quiet, meaningful (DESIGN.md MOTION dial 4). Ease-out
// exponential curves, no bounce. Framer Motion already honors the OS
// reduced-motion setting when MotionConfig reducedMotion="user" wraps the app
// (see App.tsx); these presets keep the vocabulary consistent.

import type { Transition, Variants } from 'framer-motion'

// ease-out-expo-ish
export const easeOut: Transition['ease'] = [0.16, 1, 0.3, 1]

export const spring: Transition = { type: 'spring', stiffness: 180, damping: 24 }

export const numberSpring: Transition = { type: 'spring', stiffness: 90, damping: 18 }

export const fadeUp: Variants = {
  hidden: { opacity: 0, y: 12 },
  show: { opacity: 1, y: 0, transition: { duration: 0.5, ease: easeOut } },
}

// Parent that staggers its children (meal history list).
export const stagger: Variants = {
  hidden: {},
  show: { transition: { staggerChildren: 0.06, delayChildren: 0.04 } },
}

export const scaleIn: Variants = {
  hidden: { opacity: 0, scale: 0.96 },
  show: { opacity: 1, scale: 1, transition: { duration: 0.4, ease: easeOut } },
}
