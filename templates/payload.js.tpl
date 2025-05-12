(function(){
  fetch("{{.CF_URL}}/webhook", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      event: "font-loaded",
      timestamp: {{.EVENT_TIME}},
      attackerIp: "{{.ATTACKER_IP}}",
      userAgent: navigator.userAgent,
      url: window.location.href
    })
  });
})();
