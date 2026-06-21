// A number that springs to its value, the one motion that earns its keep
// (DESIGN.md): it explains change when macros update. tabular-nums keeps the
// width stable. Honors reduced motion via the app-level MotionConfig.

import { useEffect } from 'react'
import { animate, useMotionValue, useTransform, motion } from 'framer-motion'
import { numberSpring } from '@/lib/motion'

interface Props {
  value: number
  /** decimals to render */
  decimals?: number
  className?: string
}

export function AnimatedNumber({ value, decimals = 0, className }: Props) {
  const mv = useMotionValue(value)
  const text = useTransform(mv, (v) =>
    new Intl.NumberFormat(undefined, {
      maximumFractionDigits: decimals,
      minimumFractionDigits: decimals,
    }).format(v),
  )

  useEffect(() => {
    const controls = animate(mv, value, numberSpring)
    return controls.stop
  }, [mv, value])

  return <motion.span className={`tnum ${className ?? ''}`}>{text}</motion.span>
}
