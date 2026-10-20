document.addEventListener("toggle", function (event) {
  if (!event.target.matches(".target-dropdown")) {
    return;
  }

  if (!event.target.open) {
    return;
  }

  document.querySelectorAll(".target-dropdown[open]").forEach(function (other) {
    if (other !== event.target) {
      other.open = false;
    }
  });
}, true);

// The balance spoiler re-fetches fresh data from its handle on every open —
// nothing is cached, so closing and opening it again always pulls new numbers.
document.addEventListener("toggle", function (event) {
  if (!event.target.matches(".budget-dropdown") || !event.target.open) {
    return;
  }

  var content = event.target.querySelector(".budget-content");
  var url = event.target.getAttribute("data-budget-url");

  content.innerHTML = '<span class="hint">загрузка…</span>';

  fetch(url, { credentials: "same-origin" })
    .then(function (response) { return response.text(); })
    .then(function (html) { content.innerHTML = html; })
    .catch(function () { content.innerHTML = '<div class="error">не удалось загрузить</div>'; });
}, true);

function showErrorPopup(message) {
  var dialog = document.getElementById("error-popup");
  dialog.querySelector(".popup-message").textContent = message;
  dialog.showModal();
}

document.addEventListener("click", function (event) {
  if (event.target.matches(".popup-close")) {
    event.target.closest("dialog").close();
  }
});

document.querySelectorAll(".target-select").forEach(function (select) {
  var lastValue = select.value;

  select.addEventListener("change", function () {
    var newValue = select.value;
    var body = new URLSearchParams({ scope: select.dataset.scope, target: newValue });
    if (select.dataset.squadId) {
      body.set("squad_id", select.dataset.squadId);
    }

    fetch("/panel/target", { method: "POST", body: body, credentials: "same-origin" })
      .then(function (response) {
        if (!response.ok) {
          return response.text().then(function (text) {
            select.value = lastValue;
            showErrorPopup(text);
          });
        }

        lastValue = newValue;
      })
      .catch(function () {
        select.value = lastValue;
        showErrorPopup("Сетевая ошибка");
      });
  });
});
