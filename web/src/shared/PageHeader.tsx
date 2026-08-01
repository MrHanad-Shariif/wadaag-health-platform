import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { ChevronRight } from 'lucide-react'

export interface Crumb {
  label: string
  to?: string
}

interface PageHeaderProps {
  eyebrow?: string
  breadcrumb?: Crumb[]
  title: string
  description?: string
  actions?: ReactNode
}

// The consistent header every page opens with: either a section eyebrow or
// a breadcrumb trail (never both), a title, an optional description, and a
// right-aligned actions slot for the page's primary button(s).
export function PageHeader({ eyebrow, breadcrumb, title, description, actions }: PageHeaderProps) {
  return (
    <div className="page-header">
      <div className="page-header__main">
        {breadcrumb ? (
          <nav className="breadcrumb" aria-label="Breadcrumb">
            {breadcrumb.map((crumb, i) => (
              <span key={i} className="breadcrumb__item">
                {i > 0 && <ChevronRight size={13} aria-hidden="true" className="breadcrumb__separator" />}
                {crumb.to ? <Link to={crumb.to}>{crumb.label}</Link> : <span>{crumb.label}</span>}
              </span>
            ))}
          </nav>
        ) : eyebrow ? (
          <p className="page-eyebrow">{eyebrow}</p>
        ) : null}
        <h1>{title}</h1>
        {description && <p className="page-subtitle">{description}</p>}
      </div>
      {actions && <div className="page-header__actions">{actions}</div>}
    </div>
  )
}
