import { useEffect, useState } from 'react'
import { api, can, type Me } from './api'

// Navigation is DERIVED from /api/v1/me. There is no role check anywhere in the UI: an entry
// appears when the capability it names is held, and the server refuses the same capability on the
// route, so a control that renders but is refused cannot exist.
const NAV: { group: string; label: string; path: string; capability: string }[] = [
  { group: 'Tickets', label: 'My tickets', path: '/tickets/mine', capability: 'ticket.read.own' },
  { group: 'Tickets', label: 'Queue', path: '/tickets', capability: 'ticket.read' },
  { group: 'Devices', label: 'Chromebooks', path: '/devices', capability: 'device.read' },
  { group: 'People', label: 'Students', path: '/students', capability: 'student.read' },
  { group: 'Admin', label: 'Roles', path: '/admin/roles', capability: 'roles.bind' },
]

export default function App() {
  const [me, setMe] = useState<Me | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api<Me>('/api/v1/me').then(setMe).catch(e => setError(String(e.message)))
  }, [])

  const groups = new Map<string, typeof NAV>()
  for (const n of NAV) if (can(me, n.capability)) groups.set(n.group, [...(groups.get(n.group) ?? []), n])

  return (
    <div className="shell">
      <aside>
        <h1>TightShip</h1>
        {me?.impersonating && <p className="banner">Acting as {me.effective}. Read-only.</p>}
        {[...groups.entries()].map(([g, items]) => (
          <section key={g}>
            <h2>{g}</h2>
            {items.map(i => <a key={i.path} href={i.path}>{i.label}</a>)}
          </section>
        ))}
        {me && groups.size === 0 && <p className="muted">No tools are assigned to {me.effective}.</p>}
      </aside>
      <main>
        {error && <p className="error">{error}</p>}
        {!me && !error && <p className="muted">Loading…</p>}
        {me && <p className="muted">Signed in as {me.real}. Every entry on the left is a capability you hold.</p>}
      </main>
    </div>
  )
}
