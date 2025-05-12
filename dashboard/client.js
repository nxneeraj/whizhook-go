// client.js — WebSocket client for live log streaming
(function() {
  const protocol = location.protocol === 'https:' ? 'wss' : 'ws';
  const socket = new WebSocket(`${protocol}://${location.host}/ws`);
  const logsDiv = document.getElementById('logs');

  socket.addEventListener('message', event => {
    const entry = event.data;
    const line = document.createElement('div');
    line.textContent = entry;
    logsDiv.appendChild(line);
    logsDiv.scrollTop = logsDiv.scrollHeight;
  });

  socket.addEventListener('open', () => console.log('🛰️ Connected to dashboard'));
  socket.addEventListener('close', () => console.warn('🔌 Disconnected from dashboard'));
})();
