import type { ComponentProps, ComponentType, ReactNode } from 'react'

export interface Brand {
  label: string
  homeHref: string
  mark?: ReactNode
}
export type LinkProps = Omit<ComponentProps<'a'>, 'href' | 'children'> & {
  href: string
  children: ReactNode
}
export type LinkComponent = ComponentType<LinkProps>
export interface ShellProps {
  brand: Brand
  repositoryUrl: string
  children: ReactNode
  projectActions?: ReactNode
  identity?: ReactNode
  mainId?: string
  linkComponent?: LinkComponent
}
export interface ApplicationShellProps extends ShellProps {
  businessNavigation?: ReactNode
  contextSidebar?: ReactNode
  serviceNotice?: ReactNode
}
