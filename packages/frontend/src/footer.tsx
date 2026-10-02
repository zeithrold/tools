'use client'

import { useFrontendPreferences } from './context.js'
import { shellMessages } from './messages.js'

export function SiteFooter({ repositoryUrl }: { repositoryUrl: string }): React.JSX.Element {
  if (!/^https:\/\/github\.com\/[\w.-]+\/[\w.-]+\/?$/u.test(repositoryUrl)) {
    throw new Error('SiteFooter requires the current project GitHub repository URL')
  }
  const { preferences } = useFrontendPreferences()
  const messages = shellMessages(preferences.locale)
  return (
    <footer className="ztd-footer">
      <div className="ztd-chrome">
        <span>© Zeithrold</span>
        <div className="ztd-footer-links">
          <a href={repositoryUrl} aria-label={messages.source}>GitHub</a>
          <a href="mailto:hello@ztd.me">hello@ztd.me</a>
        </div>
      </div>
    </footer>
  )
}
