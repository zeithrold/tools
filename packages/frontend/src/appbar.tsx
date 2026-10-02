'use client'

import type { ShellProps } from './shell-types.js'
import { AppearanceMenu } from './appearance.js'
import { useFrontendPreferences } from './context.js'
import { NativeLink } from './link.js'
import { LocaleSelect } from './locale-select.js'
import { shellMessages } from './messages.js'

export function Appbar(props: Omit<ShellProps, 'children' | 'repositoryUrl'>): React.JSX.Element {
  if (!props.brand.homeHref.startsWith('/') || props.brand.homeHref.startsWith('//')) {
    throw new Error('Brand homeHref must be a path within the current project')
  }
  const Link = props.linkComponent ?? NativeLink
  const { preferences } = useFrontendPreferences()
  const messages = shellMessages(preferences.locale)
  return (
    <>
      <a className="ztd-skip" href={`#${props.mainId ?? 'ztd-main'}`}>{messages.skip}</a>
      <header className="ztd-appbar">
        <div className="ztd-chrome">
          <Link className="ztd-brand" href={props.brand.homeHref}>
            {props.brand.mark ?? null}
            <span>{props.brand.label}</span>
          </Link>
          <div className="ztd-actions">
            {props.projectActions ?? null}
            <AppearanceMenu />
            <LocaleSelect />
            {props.identity ?? null}
          </div>
        </div>
      </header>
    </>
  )
}
