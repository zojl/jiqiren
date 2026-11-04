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

      summary.textContent = button.dataset.targetIcon;
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

// Budget page — "live" block: seeded server-side from the last known
// snapshot, then (only while viewing today) refreshed from the actual-data
// handle once on load and again on every click of its refresh button.
(function () {
  var block = document.querySelector("[data-budget-live]");
  if (!block) {
    return;
  }

  function applyActual(data) {
    if (data.error) {
      return;
    }

    block.querySelector('[data-field="budget"]').textContent = data.budget;
    block.querySelector('[data-field="incomeToday"]').textContent = data.incomeToday;
    block.querySelector('[data-field="incomeTotal"]').textContent = data.incomeTotal;
    block.querySelector('[data-field="contributionsToday"]').textContent = data.contributionsToday;
    block.querySelector('[data-field="contributionsTotal"]').textContent = data.contributionsTotal;

    var diffField = block.querySelector('[data-field="incomeDiff"]');
    if (diffField && block.getAttribute("data-has-diff-base") === "true") {
      var base = parseInt(block.getAttribute("data-diff-base"), 10) || 0;
      var diff = data.incomeToday - base;
      diffField.textContent = "(" + (diff >= 0 ? "+" : "") + diff + ")";
    }
  }

  function refreshActual() {
    var url = block.getAttribute("data-actual-url");
    fetch(url, { credentials: "same-origin" })
      .then(function (response) { return response.json(); })
      .then(applyActual)
      .catch(function () {});
  }

  var refreshButton = block.querySelector(".budget-refresh");
  if (refreshButton) {
    refreshButton.addEventListener("click", refreshActual);
  }

  if (block.getAttribute("data-live") === "true") {
    refreshActual();
  }
})();

// Budget page — chart block: fetches the fixed 24h window of snapshots and
// draws a plain inline SVG line chart (no charting library in this project).
// The x-axis always spans the full window regardless of how many points
// exist yet; the y-axis auto-scales to whichever series are toggled on.
(function () {
  var block = document.querySelector(".budget-chart-block");
  if (!block) {
    return;
  }

  var legend = block.querySelector(".budget-chart-legend");
  var canvas = block.querySelector(".budget-chart-canvas");

  var SERIES = [
    { key: "incomeToday", label: "доход сегодня", color: "#39ff8f", on: true },
    { key: "contributionsToday", label: "донат сегодня", color: "#e8c547", on: true },
    { key: "budget", label: "бюджет", color: "#5f8fff", on: false },
    { key: "incomeTotal", label: "доход всего", color: "#5fffe0", on: false },
    { key: "contributionsTotal", label: "донат всего", color: "#ff5fe0", on: false }
  ];

  function buildLegend() {
    SERIES.forEach(function (series) {
      var label = document.createElement("label");

      var checkbox = document.createElement("input");
      checkbox.type = "checkbox";
      checkbox.checked = series.on;
      checkbox.addEventListener("change", function () {
        series.on = checkbox.checked;
        render();
      });

      var swatch = document.createElement("span");
      swatch.className = "swatch";
      swatch.style.background = series.color;

      label.appendChild(checkbox);
      label.appendChild(swatch);
      label.appendChild(document.createTextNode(series.label));
      legend.appendChild(label);
    });
  }

  var data = null;

  function render() {
    if (!data || !data.points) {
      return;
    }

    var width = 760;
    var height = 280;
    var padding = { top: 10, right: 10, bottom: 20, left: 40 };

    var from = new Date(data.from).getTime();
    var to = new Date(data.to).getTime();
    var active = SERIES.filter(function (series) { return series.on; });

    var points = data.points.map(function (point) {
      return { t: new Date(point.capturedAt).getTime(), kind: point.kind, raw: point };
    });

    var values = [];
    points.forEach(function (point) {
      active.forEach(function (series) {
        values.push(point.raw[series.key]);
      });
    });

    var minValue = values.length ? Math.min.apply(null, values) : 0;
    var maxValue = values.length ? Math.max.apply(null, values) : 1;
    if (minValue === maxValue) {
      maxValue = minValue + 1;
    }

    function x(t) {
      return padding.left + (t - from) / (to - from) * (width - padding.left - padding.right);
    }

    function y(value) {
      return height - padding.bottom - (value - minValue) / (maxValue - minValue) * (height - padding.top - padding.bottom);
    }

    var svg = '<svg viewBox="0 0 ' + width + ' ' + height + '" preserveAspectRatio="none">';
    svg += '<line x1="' + padding.left + '" y1="' + (height - padding.bottom) + '" x2="' + (width - padding.right) + '" y2="' + (height - padding.bottom) + '" stroke="#1d2e20" />';

    active.forEach(function (series) {
      var coords = points.map(function (point) {
        return x(point.t) + "," + y(point.raw[series.key]);
      });
      svg += '<polyline points="' + coords.join(" ") + '" fill="none" stroke="' + series.color + '" stroke-width="2" />';

      points.forEach(function (point) {
        if (point.kind === "interval") {
          return;
        }
        svg += '<circle cx="' + x(point.t) + '" cy="' + y(point.raw[series.key]) + '" r="3" fill="' + series.color + '" />';
      });
    });

    svg += "</svg>";
    canvas.innerHTML = svg;
  }

  buildLegend();

  fetch(block.getAttribute("data-chart-url"), { credentials: "same-origin" })
    .then(function (response) { return response.json(); })
    .then(function (json) {
      if (json.error) {
        canvas.innerHTML = '<div class="error">' + json.error + "</div>";
        return;
      }

      data = json;
      render();
    })
    .catch(function () {
      canvas.innerHTML = '<div class="error">не удалось загрузить график</div>';
    });
})();
