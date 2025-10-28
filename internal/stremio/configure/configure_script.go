package configure

import "html/template"

func GetScriptStoreTokenDescription(nameSelector, tokenSelector string) template.JS {
	if nameSelector == "" {
		nameSelector = "'[data-field-store-token]'"
	}
	if tokenSelector == "" {
		tokenSelector = "`[name='${nameField.dataset.fieldStoreToken}']`"
	}
	return template.JS(`
function onStoreNameChangeUpdateStoreTokenDescription(nameField) {
  if (!nameField || !nameField.options[nameField.selectedIndex].textContent) {
    return;
  }
  const storeFallback = {
    alldebrid: "ad",
    debrider: "dr",
    debridlink: "dl",
    easydebrid: "ed",
    offcloud: "oc",
    pikpak: "pp",
    premiumize: "pm",
    realdebrid: "rd",
    torbox: "tb",
    p2p: "p2p",
  };
  const nameDescElem = document.querySelector(` + "`[name='${nameField.name}'] + small > span.description`" + `);
  if (nameDescElem) {
    const descByStore = {
      "*": "…",
      "": "Use <a target='_blank' href='https://github.com/MunifTanjim/stremthru?tab=readme-ov-file#stremthru_store_auth'><code>STREMTHRU_STORE_AUTH</code></a> config",
      ad: "<a type='button' class='outline mb-0' style='font-size: 0.75rem; padding: 0.02em 0.5em;' target='_blank' href='https://alldebrid.com'>Sign Up</a>",
      dr: "<a type='button' class='outline mb-0' style='font-size: 0.75rem; padding: 0.02em 0.5em;' target='_blank' href='https://debrider.app'>Sign Up</a>",
      dl: "<a type='button' class='outline mb-0' style='font-size: 0.75rem; padding: 0.02em 0.5em;' target='_blank' href='https://debrid-link.com/id/4v8Uc'>Sign Up</a>",
      ed: "<a type='button' class='outline mb-0' style='font-size: 0.75rem; padding: 0.02em 0.5em;' target='_blank' href='https://paradise-cloud.com/products/easydebrid'>Sign Up</a>",
      oc: "<a type='button' class='outline mb-0' style='font-size: 0.75rem; padding: 0.02em 0.5em;' target='_blank' href='https://offcloud.com/?=ce30ae1f'>Sign Up</a>",
      pm: "<a type='button' class='outline mb-0' style='font-size: 0.75rem; padding: 0.02em 0.5em;' target='_blank' href='https://www.premiumize.me/ref/634502061'>Sign Up</a>",
      pp: "<a type='button' class='outline mb-0' style='font-size: 0.75rem; padding: 0.02em 0.5em;' target='_blank' href='https://mypikpak.com/drive/activity/invited?invitation-code=46013321'>Sign Up</a> Invitation Code: <a target='_blank' href='https://mypikpak.com/drive/activity/invited?invitation-code=46013321'><code>46013321</code></a>",
      rd: "<a type='button' class='outline mb-0' style='font-size: 0.75rem; padding: 0.02em 0.5em;' target='_blank' href='http://real-debrid.com/?id=12448969'>Sign Up<a>",
      tb: "<a type='button' class='outline mb-0' style='font-size: 0.75rem; padding: 0.02em 0.5em;' target='_blank' href='https://torbox.app/subscription?referral=fbe2c844-4b50-416a-9cd8-4e37925f5dfa'>Sign Up</a> Referral Code: <a target='_blank' href='https://torbox.app/subscription?referral=fbe2c844-4b50-416a-9cd8-4e37925f5dfa'><code>fbe2c844-4b50-416a-9cd8-4e37925f5dfa</code></a>",
      p2p: "⚠️ Peer-to-Peer (🧪 Experimental)",
    };
    nameDescElem.innerHTML = descByStore[nameField.value] || descByStore[storeFallback[nameField.value]] || descByStore["*"] || "";
  }
	const tokenField = document.querySelector(` + tokenSelector + `)
  const tokenDescElem = document.querySelector(` + "`[name='${tokenField.name}'] + small > span.description`" + `);
  if (tokenDescElem) {
		const descByStore = {
			"*": "API Key",
			"": "StremThru Basic Auth Token (base64 encoded) from <a href='https://github.com/MunifTanjim/stremthru?tab=readme-ov-file#configuration' target='_blank'><code>STREMTHRU_PROXY_AUTH</code></a>",
			ad: "AllDebrid <a href='https://alldebrid.com/apikeys' target='_blank'>API Key</a>",
			dr: "Debrider <a href='https://debrider.app/dashboard/account#:~:text=Use%20EasyDebrid-,API%20Key,-Generate%20a%20secret' target='_blank'>API Key</a>",
			dl: "DebridLink <a href='https://debrid-link.com/webapp/apikey' target='_blank'>API Key</a>",
			ed: "EasyDebrid <a href='https://paradise-cloud.com/guides/easydebrid-api-key' target='_blank'>API Key</a>",
			oc: "Offcloud <a href='https://offcloud.com/#/account' target='_blank'>credential</a> in <code>email:password</code> format, e.g. <code>john.doe@example.com:secret-password</code>",
			pm: "Premiumize <a href='https://www.premiumize.me/account' target='_blank'>API Key</a>",
			pp: "PikPak <a href='https://mypikpak.com/drive/account/basic' target='_blank'>credential</a> in <code>email:password</code> format, e.g. <code>john.doe@example.com:secret-password</code>",
			rd: "RealDebrid <a href='https://real-debrid.com/apitoken' target='_blank'>API Token</a>",
			tb: "TorBox <a href='https://torbox.app/settings' target='_blank'>API Key</a>",
			p2p: "…",
		};
    tokenDescElem.innerHTML = descByStore[nameField.value] || descByStore[storeFallback[nameField.value]] || descByStore["*"] || "";
    tokenField.disabled = nameField.value === "p2p";
    if (nameField.value === "p2p") {
      tokenField.value = "";
    }
  }
}

document.querySelectorAll(` + nameSelector + `).forEach((nameField) => {
	onStoreNameChangeUpdateStoreTokenDescription(nameField);
	nameField.addEventListener("change", (e) => {
		onStoreNameChangeUpdateStoreTokenDescription(e.target);
	})
});
`)
}
