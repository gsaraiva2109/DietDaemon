// Energy-source split donut: where today's calories come from (protein/carbs/
// fat, 4/4/9 kcal per gram). Center shows total kcal. Legend pairs color with
// label + percent (never color alone).

import type { ComponentProps } from 'react'
import { Pie, PieChart, ResponsiveContainer, Sector } from 'recharts'
import { useTranslation } from 'react-i18next'
import type { Macros } from '@/lib/types'
import { cssVar } from '@/lib/format'

const fallbackSliceColor = cssVar('--color-line')

type MacroSliceShapeProps = ComponentProps<typeof Sector> & {
  payload?: { color?: string }
}

function MacroSliceShape(props: Readonly<MacroSliceShapeProps>) {
  return <Sector {...props} fill={props.payload?.color ?? fallbackSliceColor} stroke="none" />
}

export function MacroDonut({ consumed }: Readonly<{ consumed: Macros }>) {
  const { t } = useTranslation()
  const slices = [
    { key: 'Protein', kcal: consumed.Protein * 4, color: cssVar('--color-protein') },
    { key: 'Carbs', kcal: consumed.Carbs * 4, color: cssVar('--color-carbs') },
    { key: 'Fat', kcal: consumed.Fat * 9, color: cssVar('--color-fat') },
  ]
  const total = slices.reduce((s, x) => s + x.kcal, 0)
  const data = total > 0 ? slices : [{ key: 'none', kcal: 1, color: cssVar('--color-line') }]

  return (
    <div className="flex items-center gap-5">
      <div className="relative size-32 shrink-0">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={data}
              dataKey="kcal"
              innerRadius={42}
              outerRadius={62}
              paddingAngle={total > 0 ? 3 : 0}
              stroke="none"
              shape={MacroSliceShape}
            />
          </PieChart>
        </ResponsiveContainer>
        <div className="absolute inset-0 grid place-items-center text-center">
          <div>
            <div className="text-lg font-bold leading-none text-ink tnum">{Math.round(total)}</div>
            <div className="text-[10px] uppercase tracking-[0.12em] text-muted">kcal</div>
          </div>
        </div>
      </div>
      <ul className="flex flex-col gap-2">
        {slices.map((s) => (
          <li key={s.key} className="flex items-center gap-2 text-sm">
            <span className="size-2.5 rounded-full" style={{ backgroundColor: s.color }} />
            <span className="font-medium text-ink">{t(`macroDonut.macro.${s.key}`)}</span>
            <span className="text-muted tnum">{total > 0 ? Math.round((s.kcal / total) * 100) : 0}%</span>
          </li>
        ))}
      </ul>
    </div>
  )
}
