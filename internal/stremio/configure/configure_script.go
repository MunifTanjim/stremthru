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
	const tokenField = document.querySelector(` + tokenSelector + `)
  const tokenDescElem = document.querySelector(` + "`[name='${tokenField.name}'] + small > span.description`" + `);
  if (tokenDescElem) {
		const descByStore = {
			"*": "API Key",
			"": "StremThru Basic Auth Token (base64 encoded) from <a href='https://github.com/MunifTanjim/stremthru?tab=readme-ov-file#configuration' target='_blank'><code>STREMTHRU_PROXY_AUTH</code></a>",
			ad: "AllDebrid <a href='https://alldebrid.com/apikeys' target='_blank'>API Key</a>",
			dl: "DebridLink <a href='https://debrid-link.com/webapp/apikey' target='_blank'>API Key</a>",
			ed: "EasyDebrid <a href='https://paradise-cloud.com/guides/easydebrid-api-key' target='_blank'>API Key</a>",
			oc: "Offcloud <a href='https://offcloud.com/#/account' target='_blank'>credential</a> in <code>email:password</code> format, e.g. <code>john.doe@example.com:secret-password</code>",
			pm: "Premiumize <a href='https://www.premiumize.me/account' target='_blank'>API Key</a>",
			pp: "PikPak <a href='https://mypikpak.com/drive/account/basic' target='_blank'>credential</a> in <code>email:password</code> format, e.g. <code>john.doe@example.com:secret-password</code>",
			rd: "RealDebrid <a href='https://real-debrid.com/apitoken' target='_blank'>API Token</a>",
			tb: "TorBox <a href='https://torbox.app/settings' target='_blank'>API Key</a>",
		};
		const storeFallback = {
			alldebrid: "ad",
			debridlink: "dl",
			easydebrid: "ed",
			offcloud: "oc",
			pikpak: "pp",
			premiumize: "pm",
			realdebrid: "rd",
			torbox: "tb",
		};
    tokenDescElem.innerHTML = descByStore[nameField.value] || descByStore[storeFallback[nameField.value]] || descByStore["*"] || "";
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
