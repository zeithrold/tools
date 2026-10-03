export function FontSpecimens(): React.JSX.Element {
  return (
    <section className="font-specimens" aria-labelledby="fonts-heading">
      <h1 id="fonts-heading">Noto typography verification</h1>
      <p>English and CJK samples use Google Fonts and language-specific glyph forms.</p>
      <p id="font-en" lang="en">English typography ABC xyz 0123456789 Café</p>
      <p id="font-zh" lang="zh-CN">中文简体汉字，外观与语言。</p>
      <p id="font-ja" lang="ja">日本語 ひらがな カタカナ</p>
      <p id="font-ko" lang="ko">한국어 한글 글꼴</p>
      <p id="font-bold" lang="zh-CN"><strong>English 中文 日本語 한국어 600</strong></p>
      <p id="font-mixed" lang="en">Noto emoji 😀 with English 0123</p>
      <p>Color Noto Emoji, including composed sequences:</p>
      <div className="emoji-specimens">
        <span id="emoji-face" className="ztd-emoji" role="img" aria-label="Smiling face">😀</span>
        <span id="emoji-heart" className="ztd-emoji" role="img" aria-label="Heart">❤️</span>
        <span id="emoji-technologist" className="ztd-emoji" role="img" aria-label="Woman technologist">👩🏽‍💻</span>
        <span id="emoji-family" className="ztd-emoji" role="img" aria-label="Family">👨‍👩‍👧‍👦</span>
        <span id="emoji-rainbow" className="ztd-emoji" role="img" aria-label="Rainbow flag">🏳️‍🌈</span>
        <span id="emoji-flag" className="ztd-emoji" role="img" aria-label="Japanese flag">🇯🇵</span>
      </div>
    </section>
  )
}
