import { ArrowRight, ArrowUpRight, Sparkles } from 'lucide-react'

export function ReviewNavigation(): React.JSX.Element {
  return (
    <nav className="review-navigation" aria-label="Studio">
      <a href="#collection">Collection <span>02</span></a>
      <a href="https://example.org">About <ArrowUpRight size={16} strokeWidth={2} aria-hidden="true" /></a>
    </nav>
  )
}

export function ReviewContent(): React.JSX.Element {
  return (
    <div className="review-content">
      <section className="review-intro" aria-labelledby="review-heading">
        <div>
          <p className="review-eyebrow">
            <Sparkles size={12} strokeWidth={2} aria-hidden="true" /> Independent design collection
          </p>
          <h1 id="review-heading">Small ideas.<br /><span>Room for the details.</span></h1>
        </div>
        <p className="review-description">A quiet place to explore useful things,<br />and the spaces between them.</p>
      </section>
      <section id="collection" aria-labelledby="collection-heading">
        <div className="review-section-label"><h2 id="collection-heading">Collection</h2><span>2 studies</span></div>
        <a className="review-card" href="#materials">
          <div className="review-sample" aria-hidden="true"><span /><span /><span /></div>
          <div>
            <p className="review-eyebrow">01 / Visual study</p>
            <h3>Material notes</h3>
            <p>Simple shapes, softer edges, and a little breathing room.</p>
            <span className="review-open">Explore study <ArrowRight size={16} strokeWidth={2} aria-hidden="true" /></span>
          </div>
          <span className="review-arrow" aria-hidden="true"><ArrowUpRight size={20} strokeWidth={2} aria-hidden="true" /></span>
        </a>
        <p id="materials" className="review-caption">A synthetic consumer page for reviewing shared chrome and tokens.</p>
      </section>
    </div>
  )
}
