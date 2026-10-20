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

document.addEventListener("toggle", function (event) {
  if (!event.target.matches(".target-picker") || !event.target.open) {
    return;
  }

  document.querySelectorAll(".target-picker[open]").forEach(function (other) {
    if (other !== event.target) {
      other.open = false;
    }
  });
}, true);

document.addEventListener("click", function (event) {
  document.querySelectorAll(".target-picker[open]").forEach(function (picker) {
    if (!picker.contains(event.target)) {
      picker.open = false;
    }
  });
});

document.addEventListener("click", function (event) {
  var button = event.target.closest(".target-option");
  if (!button) {
    return;
  }

  var picker = button.closest(".target-picker");
  var summary = picker.querySelector("summary");
  var body = new URLSearchParams({ scope: picker.dataset.scope, target: button.dataset.value });
  if (picker.dataset.squadId) {
    body.set("squad_id", picker.dataset.squadId);
  }

  fetch("/panel/target", { method: "POST", body: body, credentials: "same-origin" })
    .then(function (response) {
      if (!response.ok) {
        return response.text().then(function (text) { showErrorPopup(text); });
      }

      summary.textContent = button.dataset.icon;
      picker.querySelectorAll(".target-option").forEach(function (option) {
        option.classList.toggle("selected", option === button);
      });
    })
    .catch(function () {
      showErrorPopup("Сетевая ошибка");
    })
    .finally(function () {
      picker.open = false;
    });
});
