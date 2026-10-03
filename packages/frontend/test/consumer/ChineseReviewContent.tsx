export function ChineseReviewContent(): React.JSX.Element {
  return (
    <div className="review-content" lang="zh-CN">
      <section className="review-intro" aria-labelledby="chinese-heading">
        <div>
          <p className="review-eyebrow">个人设计与工具集</p>
          <h1 id="chinese-heading">小小工具。<br /><span>细节，恰到好处。</span></h1>
        </div>
        <p className="review-description">探索形式与功能，<br />也在意它们之间的留白。</p>
      </section>
      <section aria-labelledby="chinese-collection">
        <div className="review-section-label"><h2 id="chinese-collection">页面目录</h2><span>一个页面</span></div>
        <div className="review-card">
          <div className="review-sample" aria-hidden="true"><span /><span /><span /></div>
          <div>
            <p>交互与实用工具</p>
            <h3>设计研究</h3>
            <p>这是独立的中文测试页面，用于测量字体加载。界面使用真实组件，内容不代表任何生产项目。</p>
          </div>
        </div>
      </section>
    </div>
  )
}
