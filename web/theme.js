export function initTheme() {
  const saved = localStorage.getItem('theme') || 'dark';
  document.documentElement.setAttribute('data-theme', saved);

  window.setTheme = function(val) {
    document.documentElement.setAttribute('data-theme', val);
    localStorage.setItem('theme', val);

    // Update theme-color meta tag
    const meta = document.querySelector('meta[name="theme-color"]');
    if(meta) {
        const bg = getComputedStyle(document.body).getPropertyValue('--bg').trim();
        meta.setAttribute('content', bg);
    }
  };

  // Initial meta tag update
  const pkg = getComputedStyle(document.body).getPropertyValue('--bg').trim();
  const meta = document.querySelector('meta[name="theme-color"]');
  if(meta) meta.setAttribute('content', pkg);
}
