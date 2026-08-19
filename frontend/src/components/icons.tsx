import type { SVGProps } from 'react'

export type IconName =
  | 'activity'
  | 'bell'
  | 'chevron-down'
  | 'chevron-left'
  | 'chevron-right'
  | 'close'
  | 'code'
  | 'dashboard'
  | 'environment'
  | 'flag'
  | 'key'
  | 'logout'
  | 'menu'
  | 'project'
  | 'search'
  | 'settings'
  | 'users'

interface IconProps extends SVGProps<SVGSVGElement> {
  name: IconName
  size?: number
}

const paths: Record<IconName, string[]> = {
  activity: ['M3 12h4l2-5 4 10 2-5h6'],
  bell: [
    'M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9',
    'M10 21h4',
  ],
  'chevron-down': ['m6 9 6 6 6-6'],
  'chevron-left': ['m15 18-6-6 6-6'],
  'chevron-right': ['m9 18 6-6-6-6'],
  close: ['M6 6l12 12M18 6 6 18'],
  code: ['m8 9-4 3 4 3M16 9l4 3-4 3M14 5l-4 14'],
  dashboard: [
    'M3 3h7v7H3z',
    'M14 3h7v7h-7z',
    'M3 14h7v7H3z',
    'M14 14h7v7h-7z',
  ],
  environment: [
    'm12 3 8 4.5-8 4.5-8-4.5L12 3z',
    'm4 12 8 4.5 8-4.5',
    'm4 16.5 8 4.5 8-4.5',
  ],
  flag: ['M5 21V4m0 0h10l-1.5 3L15 10H5'],
  key: [
    'M15.5 7.5a4.5 4.5 0 1 1-3.2 7.7L9 18.5V21H6.5v-2.5H4V16l4.3-4.3a4.5 4.5 0 0 1 7.2-4.2z',
    'M15.5 7.5h.01',
  ],
  logout: ['M10 17l5-5-5-5', 'M15 12H3', 'M14 3h4a3 3 0 0 1 3 3v12a3 3 0 0 1-3 3h-4'],
  menu: ['M4 7h16M4 12h16M4 17h16'],
  project: ['M3 7h7l2 2h9v10H3z'],
  search: ['M11 18a7 7 0 1 1 0-14 7 7 0 0 1 0 14z', 'm20 20-4-4'],
  settings: [
    'M12 15.5a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7z',
    'M19.4 15a1.7 1.7 0 0 0 .34 1.88l.06.06-1.8 3.12-.08-.02a1.7 1.7 0 0 0-1.8.16l-.72.42a1.7 1.7 0 0 0-.86 1.67V22H10.9v-.1a1.7 1.7 0 0 0-.86-1.67l-.72-.42a1.7 1.7 0 0 0-1.8-.16l-.08.02-1.8-3.12.06-.06A1.7 1.7 0 0 0 6 15l-.42-.72a1.7 1.7 0 0 0-1.47-.85H4V9.82h.1a1.7 1.7 0 0 0 1.48-.85L6 8.25a1.7 1.7 0 0 0-.3-1.88l-.06-.06 1.8-3.12.08.02a1.7 1.7 0 0 0 1.8-.16l.72-.42A1.7 1.7 0 0 0 10.9.96V.88h3.6v.08a1.7 1.7 0 0 0 .86 1.67l.72.42a1.7 1.7 0 0 0 1.8.16l.08-.02 1.8 3.12-.06.06a1.7 1.7 0 0 0-.3 1.88l.42.72a1.7 1.7 0 0 0 1.47.85h.1v3.62h-.1a1.7 1.7 0 0 0-1.47.85L19.4 15z',
  ],
  users: [
    'M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2',
    'M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8z',
    'M22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75',
  ],
}

export function Icon({ name, size = 20, ...props }: IconProps) {
  return (
    <svg
      aria-hidden="true"
      fill="none"
      height={size}
      viewBox="0 0 24 24"
      width={size}
      stroke="currentColor"
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth="1.8"
      {...props}
    >
      {paths[name].map((path) => (
        <path d={path} key={path} />
      ))}
    </svg>
  )
}
