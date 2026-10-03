// Browser-side analysis adapter. Input is CSS captured from real, completed stylesheet responses.
export function installAnalysisCssReader({ styleNonce, sheets }) {
  const originalCreate = Document.prototype.createElement
  const OriginalRequest = window.XMLHttpRequest
  const snapshot = new Map(sheets)
  Document.prototype.createElement = function (name, ...args) {
    const element = originalCreate.call(this, name, ...args)
    if (name.toLowerCase() === 'style') {
      element.nonce = styleNonce
    }
    return element
  }
  // axe-core 4.13 uses XHR for its CSSOM reader, rather than fetch.
  window.XMLHttpRequest = class AnalysisRequest extends OriginalRequest {
    #cachedBody

    open(method, url, ...args) {
      this.#cachedBody = method === 'GET' ? snapshot.get(String(url)) : undefined
      return super.open(method, url, ...args)
    }

    get responseText() {
      return this.#cachedBody ?? super.responseText
    }

    send(...args) {
      if (this.#cachedBody !== undefined) {
        queueMicrotask(() => this.dispatchEvent(new ProgressEvent('loadend', { loaded: this.#cachedBody.length })))
        return
      }
      super.send(...args)
    }
  }
  return () => {
    Document.prototype.createElement = originalCreate
    window.XMLHttpRequest = OriginalRequest
  }
}
