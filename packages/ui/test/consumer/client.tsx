import type { FixtureProps } from './Fixture.js'
import { hydrateRoot } from 'react-dom/client'
import { Fixture } from './Fixture.js'
import '@ztd-me/ui/styles.css'
import './fixture.css'
import './review.css'

const root = document.querySelector('#root')
const payload = document.querySelector('#initial')?.textContent
if (!root || !payload) throw new Error('Fixture requires SSR payload')
const props = JSON.parse(payload) as FixtureProps
hydrateRoot(root, <Fixture {...props} />)
