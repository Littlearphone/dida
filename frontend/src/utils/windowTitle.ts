/**
 * 设置桌面窗口标题
 *
 * 生产环境（WebView2）通过 Go 绑定的 setWindowTitle 修改原生窗口标题；
 * 开发环境（浏览器）回退到 document.title，方便调试。
 */
export function setWindowTitle(title: string): void {
  if (typeof window !== 'undefined' && (window as any).setWindowTitle) {
    // WebView2 中通过 Go bind 修改窗口标题（实时反映到任务栏）
    (window as any).setWindowTitle(title)
  }
  // 回退：浏览器标签页或 DevTools 中也能看到
  document.title = title
}
