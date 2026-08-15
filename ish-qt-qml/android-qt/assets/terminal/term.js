(() => {
  const output = document.getElementById('output');
  const terminal = document.getElementById('terminal');
  const cursor = document.getElementById('cursor');
  let socket = null;
  let fontSize = 14;
  const decoder = new TextDecoder('utf-8', { fatal: false });

  function append(value) {
    if (value == null) return;
    output.textContent += String(value);
    terminal.scrollTop = terminal.scrollHeight;
  }

  function send(value) {
    if (!value) return;
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ type: 'input', data: value }));
    }
  }

  function connect(url) {
    if (!url || !/^wss?:/i.test(url)) return;
    try {
      socket = new WebSocket(url);
      socket.binaryType = 'arraybuffer';
      socket.onopen = () => { append('\\r\\n'); terminal.focus(); };
      socket.onmessage = event => {
        if (event.data instanceof ArrayBuffer) append(decoder.decode(new Uint8Array(event.data), { stream: true }));
        else {
          try {
            const msg = JSON.parse(event.data);
            append(msg.data ?? msg.output ?? msg.text ?? event.data);
          } catch (_) { append(event.data); }
        }
      };
      socket.onerror = () => append('\\r\\n[WebSocket error]\\r\\n');
      socket.onclose = () => append('\\r\\n[session closed]\\r\\n');
    } catch (error) { append('\\r\\n[connection error: ' + error + ']\\r\\n'); }
  }

  terminal.addEventListener('keydown', event => {
    if (event.key === 'Enter') { send('\\r'); event.preventDefault(); return; }
    if (event.key === 'Backspace') { send('\\x7f'); event.preventDefault(); return; }
    if (event.key === 'Tab') { send('\\t'); event.preventDefault(); return; }
    if (event.key === 'Escape') { send('\\x1b'); event.preventDefault(); return; }
    if (event.key === 'ArrowUp') { send('\\x1b[A'); event.preventDefault(); return; }
    if (event.key === 'ArrowDown') { send('\\x1b[B'); event.preventDefault(); return; }
    if (event.key === 'ArrowRight') { send('\\x1b[C'); event.preventDefault(); return; }
    if (event.key === 'ArrowLeft') { send('\\x1b[D'); event.preventDefault(); return; }
    if (event.ctrlKey && event.key.length === 1) { send(String.fromCharCode(event.key.toUpperCase().charCodeAt(0) - 64)); event.preventDefault(); return; }
    if (event.key.length === 1 && !event.metaKey && !event.altKey) { send(event.key); event.preventDefault(); }
  });

  window.ishSendInput = send;
  window.ishPaste = async () => { try { send(await navigator.clipboard.readText()); } catch (_) {} };
  window.ishFontStep = delta => { fontSize = Math.max(8, Math.min(36, fontSize + Number(delta || 0))); terminal.style.fontSize = fontSize + 'px'; };
  window.ishFontReset = () => { fontSize = 14; terminal.style.fontSize = '14px'; };
  window.ishClear = () => { output.textContent = ''; };
  window.ishCopy = () => { const s = window.getSelection(); if (s) navigator.clipboard?.writeText(s.toString()); };
  window.ishConnect = connect;

  append('iSH Qt terminal\\r\\n');
  append('Waiting for the native session…\\r\\n');
  terminal.focus();
  const query = new URLSearchParams(location.search);
  if (query.get('ws')) connect(query.get('ws'));
})();
