document.addEventListener("toggle", function (event) {
  if (!event.target.matches(".nav-burger") || !event.target.open) {
    return;
  }

  document.querySelectorAll(".nav-burger[open]").forEach(function (other) {
    if (other !== event.target) {
      other.open = false;
    }
  });
}, true);

document.addEventListener("click", function (event) {
  document.querySelectorAll(".nav-burger[open]").forEach(function (burger) {
    if (!burger.contains(event.target)) {
      burger.open = false;
    }
  });
});

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

// Target plan form — "recurring" fields (end date) only make sense once the
// "повторяется" radio is picked, so they're hidden until then.
document.addEventListener("change", function (event) {
  if (!event.target.matches(".target-plan-form input[name='type']")) {
    return;
  }

  var form = event.target.closest("form");
  var recurringFields = form.querySelector("[data-type-fields='recurring']");
  if (recurringFields) {
    recurringFields.hidden = event.target.value !== "recurring";
  }
});

// MSK is read as a fixed UTC+3 offset straight off the epoch, matching the
// backend's treatment of Moscow time (no DST since 2014) rather than
// whatever timezone the browser happens to be in.
function pad2(n) {
  return (n < 10 ? "0" : "") + n;
}

function mskHour(ms) {
  return (new Date(ms).getUTCHours() + 3) % 24;
}

function mskTimeLabel(ms) {
  var d = new Date(ms);
  return pad2(mskHour(ms)) + ":" + pad2(d.getUTCMinutes());
}

// Budget page — "live" block: seeded server-side from the last known
// snapshot, then (only while viewing today) refreshed from the actual-data
// handle once on load and again on every click of its refresh button.
(function () {
  var block = document.querySelector("[data-budget-live]");
  if (!block) {
    return;
  }

  function updateDiff(field, baseAttr, liveValue) {
    var diffField = block.querySelector('[data-field="' + field + '"]');
    if (!diffField || block.getAttribute("data-has-diff-base") !== "true") {
      return;
    }

    var base = parseInt(block.getAttribute(baseAttr), 10) || 0;
    var diff = liveValue - base;
    diffField.textContent = "(" + (diff >= 0 ? "+" : "") + diff + ")";
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

    updateDiff("incomeDiff", "data-income-diff-base", data.incomeToday);
    updateDiff("contributionsDiff", "data-contributions-diff-base", data.contributionsToday);

    var updatedAtField = block.querySelector('[data-field="updatedAt"]');
    if (updatedAtField) {
      updatedAtField.textContent = mskTimeLabel(Date.now()) + " МСК";
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
// Hour labels and the tooltip's time both read MSK as UTC+3 directly off
// the epoch, matching the backend's fixed-offset treatment of Moscow time
// (no DST since 2014) rather than the browser's local timezone.
(function () {
  var block = document.querySelector(".budget-chart-block");
  if (!block) {
    return;
  }

  var legend = block.querySelector(".budget-chart-legend");
  var svg = block.querySelector(".budget-chart-svg");
  var tooltip = block.querySelector(".budget-chart-tooltip");
  var canvas = block.querySelector(".budget-chart-canvas");
  var updatedAtLabel = block.querySelector(".budget-chart-updated-at");

  // The series' own hues are a fixed pair of palettes (not per-scheme) picked
  // for contrast against a light vs. a dark page background — the bright
  // night set goes washed-out and unreadable on a light/day background,
  // so pick by the mode actually in effect (resolving "auto" the same way
  // the CSS does) rather than one fixed palette for every mode.
  function isDarkMode() {
    var mode = document.documentElement.getAttribute("data-mode");
    if (mode === "night") {
      return true;
    }

    if (mode === "day") {
      return false;
    }

    return !!(window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches);
  }

  var seriesColors = isDarkMode()
    ? { incomeToday: "#39ff8f", contributionsToday: "#e8c547", budget: "#5f8fff", incomeTotal: "#ff7a5f", contributionsTotal: "#ff5fe0" }
    : { incomeToday: "#1f8a46", contributionsToday: "#a9791a", budget: "#2452b8", incomeTotal: "#b8391f", contributionsTotal: "#b8318f" };

  var SERIES = [
    { key: "incomeToday", label: "доход за день", color: seriesColors.incomeToday, on: true },
    { key: "contributionsToday", label: "донат за день", color: seriesColors.contributionsToday, on: true },
    { key: "budget", label: "бюджет", color: seriesColors.budget, on: false },
    { key: "incomeTotal", label: "доход всего", color: seriesColors.incomeTotal, on: false },
    { key: "contributionsTotal", label: "донат всего", color: seriesColors.contributionsTotal, on: false }
  ];

  // Structural lines (gridlines/guides) use currentColor off the inherited
  // --fg so they stay legible under any scheme/mode automatically; the
  // background-colored ring around hover dots and the error text need the
  // theme's actual resolved colors, read once from the live CSS variables.
  var themeStyle = getComputedStyle(document.documentElement);
  var themeBg = (themeStyle.getPropertyValue("--bg") || "#0a0d0a").trim();
  var themeDanger = (themeStyle.getPropertyValue("--danger") || "#ff5f5f").trim();

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

  var width = 760;
  var height = 280;
  var padding = { top: 10, right: 10, bottom: 28, left: 54 };
  var plotWidth = width - padding.left - padding.right;
  var plotHeight = height - padding.top - padding.bottom;

  var data = null;
  var points = [];
  var activeSeries = [];
  var scaleX = null;
  var scaleY = null;

  function render() {
    if (!data || !data.points) {
      return;
    }

    var from = new Date(data.from).getTime();
    var to = new Date(data.to).getTime();
    activeSeries = SERIES.filter(function (series) { return series.on; });

    points = data.points.map(function (point) {
      return { t: new Date(point.capturedAt).getTime(), kind: point.kind, raw: point };
    });

    var values = [];
    points.forEach(function (point) {
      activeSeries.forEach(function (series) {
        values.push(point.raw[series.key]);
      });
    });

    var minValue = 0;
    var maxValue = values.length ? Math.max.apply(null, values) : 1;
    if (maxValue === minValue) {
      maxValue = minValue + 1;
    }

    scaleX = function (t) {
      return padding.left + (t - from) / (to - from) * plotWidth;
    };
    scaleY = function (value) {
      return height - padding.bottom - (value - minValue) / (maxValue - minValue) * plotHeight;
    };

    var parts = [];

    parts.push('<line x1="' + padding.left + '" y1="' + (height - padding.bottom) + '" x2="' + (width - padding.right) + '" y2="' + (height - padding.bottom) + '" stroke="currentColor" opacity="0.3" />');

    var yTicks = 4;
    for (var i = 0; i <= yTicks; i++) {
      var value = minValue + (maxValue - minValue) * i / yTicks;
      var yy = scaleY(value);
      parts.push('<line x1="' + padding.left + '" y1="' + yy + '" x2="' + (width - padding.right) + '" y2="' + yy + '" stroke="currentColor" opacity="0.2" stroke-dasharray="2,3" />');
      parts.push('<text x="' + (padding.left - 8) + '" y="' + yy + '" text-anchor="end" dominant-baseline="middle" font-size="11" fill="currentColor" opacity="0.6">' + Math.round(value) + "</text>");
    }

    var hourMs = 60 * 60 * 1000;
    var stepMs = 3 * hourMs;
    for (var t = Math.ceil(from / stepMs) * stepMs; t <= to; t += stepMs) {
      var xx = scaleX(t);
      parts.push('<text x="' + xx + '" y="' + (height - padding.bottom + 16) + '" text-anchor="middle" font-size="11" fill="currentColor" opacity="0.6">' + pad2(mskHour(t)) + ":00</text>");
    }

    activeSeries.forEach(function (series) {
      var coords = points.map(function (point) {
        return scaleX(point.t) + "," + scaleY(point.raw[series.key]);
      });
      parts.push('<polyline points="' + coords.join(" ") + '" fill="none" stroke="' + series.color + '" stroke-width="2" />');

      points.forEach(function (point) {
        if (point.kind === "interval") {
          return;
        }
        parts.push('<circle cx="' + scaleX(point.t) + '" cy="' + scaleY(point.raw[series.key]) + '" r="3" fill="' + series.color + '" />');
      });
    });

    parts.push('<g class="hover-guide" style="display:none"><line class="hover-line" x1="0" y1="' + padding.top + '" x2="0" y2="' + (height - padding.bottom) + '" stroke="currentColor" opacity="0.5" stroke-dasharray="3,3" />' +
      activeSeries.map(function (series) {
        return '<circle class="hover-dot" data-key="' + series.key + '" r="4" fill="' + series.color + '" stroke="' + themeBg + '" stroke-width="1.5" />';
      }).join("") + "</g>");

    parts.push('<rect class="hover-capture" x="' + padding.left + '" y="' + padding.top + '" width="' + plotWidth + '" height="' + plotHeight + '" fill="transparent" />');

    svg.innerHTML = parts.join("");
    wireHover();
  }

  function findNearest(targetT) {
    if (!points.length) {
      return null;
    }

    var nearest = points[0];
    var bestDiff = Math.abs(points[0].t - targetT);
    for (var i = 1; i < points.length; i++) {
      var diff = Math.abs(points[i].t - targetT);
      if (diff < bestDiff) {
        bestDiff = diff;
        nearest = points[i];
      }
    }

    return nearest;
  }

  function wireHover() {
    var capture = svg.querySelector(".hover-capture");
    if (!capture) {
      return;
    }

    capture.addEventListener("mousemove", onHover);
    capture.addEventListener("mouseleave", onLeave);
  }

  function onHover(event) {
    var point = svg.createSVGPoint();
    point.x = event.clientX;
    point.y = event.clientY;

    var ctm = svg.getScreenCTM();
    if (!ctm) {
      return;
    }

    var local = point.matrixTransform(ctm.inverse());
    var from = new Date(data.from).getTime();
    var to = new Date(data.to).getTime();
    var targetT = from + (local.x - padding.left) / plotWidth * (to - from);

    var nearest = findNearest(targetT);
    if (!nearest) {
      return;
    }

    var guide = svg.querySelector(".hover-guide");
    guide.style.display = "";
    guide.querySelector(".hover-line").setAttribute("x1", scaleX(nearest.t));
    guide.querySelector(".hover-line").setAttribute("x2", scaleX(nearest.t));
    guide.querySelectorAll(".hover-dot").forEach(function (dot) {
      dot.setAttribute("cx", scaleX(nearest.t));
      dot.setAttribute("cy", scaleY(nearest.raw[dot.getAttribute("data-key")]));
    });

    var rows = activeSeries.map(function (series) {
      return '<div class="budget-chart-tooltip-row"><span class="swatch" style="background:' + series.color + '"></span>' + series.label + ": " + nearest.raw[series.key] + "</div>";
    }).join("");
    tooltip.innerHTML = '<div class="budget-chart-tooltip-time">' + mskTimeLabel(nearest.t) + "</div>" + rows;
    tooltip.hidden = false;

    var canvasRect = canvas.getBoundingClientRect();
    var left = event.clientX - canvasRect.left + 12;
    var top = event.clientY - canvasRect.top - 10;
    if (left + tooltip.offsetWidth > canvasRect.width) {
      left = event.clientX - canvasRect.left - tooltip.offsetWidth - 12;
    }
    tooltip.style.left = left + "px";
    tooltip.style.top = top + "px";
  }

  function onLeave() {
    var guide = svg.querySelector(".hover-guide");
    if (guide) {
      guide.style.display = "none";
    }
    tooltip.hidden = true;
  }

  function load() {
    fetch(block.getAttribute("data-chart-url"), { credentials: "same-origin" })
      .then(function (response) { return response.json(); })
      .then(function (json) {
        if (json.error) {
          svg.innerHTML = '<text x="' + (width / 2) + '" y="' + (height / 2) + '" text-anchor="middle" fill="' + themeDanger + '">' + json.error + "</text>";
          return;
        }

        data = json;
        render();

        if (updatedAtLabel) {
          if (data.points.length) {
            var lastPoint = data.points[data.points.length - 1];
            updatedAtLabel.textContent = mskTimeLabel(new Date(lastPoint.capturedAt).getTime()) + " МСК";
          } else {
            updatedAtLabel.textContent = "";
          }
        }
      })
      .catch(function () {
        svg.innerHTML = '<text x="' + (width / 2) + '" y="' + (height / 2) + '" text-anchor="middle" fill="' + themeDanger + '">не удалось загрузить график</text>';
      });
  }

  var refreshButton = block.querySelector(".budget-chart-refresh");
  if (refreshButton) {
    refreshButton.addEventListener("click", load);
  }

  buildLegend();
  load();
})();

// "Мой профиль" — experience growth chart: one series (total experience,
// already flattened across prestiges server-side), drawn as an inline SVG
// the same way as the budget chart. The y-axis spans the data's own
// min..max rather than starting at zero — growth over a year is small
// next to the absolute total, and would otherwise look flat.
(function () {
  var block = document.querySelector(".exp-chart-block");
  if (!block) {
    return;
  }

  var svg = block.querySelector(".budget-chart-svg");
  var tooltip = block.querySelector(".budget-chart-tooltip");
  var canvas = block.querySelector(".budget-chart-canvas");

  var themeStyle = getComputedStyle(document.documentElement);
  var themeBg = (themeStyle.getPropertyValue("--bg") || "#0a0d0a").trim();
  var themeAccent = (themeStyle.getPropertyValue("--accent") || "#39ff8f").trim();
  var themeDanger = (themeStyle.getPropertyValue("--danger") || "#ff5f5f").trim();

  var width = 760;
  var height = 280;
  var padding = { top: 10, right: 10, bottom: 28, left: 64 };
  var plotWidth = width - padding.left - padding.right;
  var plotHeight = height - padding.bottom - padding.top;
  var mskOffsetMs = 3 * 60 * 60 * 1000;
  var dayMs = 24 * 60 * 60 * 1000;

  var points = [];
  var from = 0;
  var to = 0;
  var scaleX = null;
  var scaleY = null;

  function message(text, color) {
    svg.innerHTML = '<text x="' + (width / 2) + '" y="' + (height / 2) + '" text-anchor="middle" fill="' + color + '" opacity="0.8">' + text + "</text>";
  }

  // Same MSK-as-UTC+3 treatment as the rest of the panel.
  function mskDate(ms) {
    return new Date(ms + mskOffsetMs);
  }

  function dateLabel(ms, withYear) {
    var d = mskDate(ms);
    var label = pad2(d.getUTCDate()) + "." + pad2(d.getUTCMonth() + 1);
    return withYear ? label + "." + String(d.getUTCFullYear()).slice(2) : label;
  }

  function fullDateLabel(ms) {
    var d = mskDate(ms);
    return pad2(d.getUTCDate()) + "." + pad2(d.getUTCMonth() + 1) + "." + d.getUTCFullYear() + " " + mskTimeLabel(ms);
  }

  function grouped(n) {
    return String(Math.round(n)).replace(/\B(?=(\d{3})+(?!\d))/g, " ");
  }

  function compact(n) {
    var abs = Math.abs(n);
    if (abs >= 1e9) {
      return (n / 1e9).toFixed(abs >= 1e10 ? 0 : 1) + "B";
    }
    if (abs >= 1e6) {
      return (n / 1e6).toFixed(abs >= 1e7 ? 0 : 1) + "M";
    }
    if (abs >= 1e3) {
      return (n / 1e3).toFixed(abs >= 1e4 ? 0 : 1) + "k";
    }
    return String(Math.round(n));
  }

  function levelLabel(point) {
    return point.level + (point.prestige ? "💎" + point.prestige : "") + " ур.";
  }

  function render() {
    if (!points.length) {
      message("пока нет данных", "currentColor");
      return;
    }

    // A single point (or several within moments of each other) would
    // collapse the axis to zero width — pad it to at least a day.
    var spanFrom = from;
    var spanTo = to;
    if (spanTo - spanFrom < dayMs) {
      spanFrom = spanTo - dayMs;
    }

    var values = points.map(function (point) { return point.total; });
    var minValue = Math.min.apply(null, values);
    var maxValue = Math.max.apply(null, values);
    var margin = (maxValue - minValue) * 0.05 || Math.max(1, maxValue * 0.01);
    minValue = Math.max(0, minValue - margin);
    maxValue = maxValue + margin;

    scaleX = function (t) {
      return padding.left + (t - spanFrom) / (spanTo - spanFrom) * plotWidth;
    };
    scaleY = function (value) {
      return height - padding.bottom - (value - minValue) / (maxValue - minValue) * plotHeight;
    };

    var parts = [];
    parts.push('<line x1="' + padding.left + '" y1="' + (height - padding.bottom) + '" x2="' + (width - padding.right) + '" y2="' + (height - padding.bottom) + '" stroke="currentColor" opacity="0.3" />');

    var yTicks = 4;
    for (var i = 0; i <= yTicks; i++) {
      var value = minValue + (maxValue - minValue) * i / yTicks;
      var yy = scaleY(value);
      parts.push('<line x1="' + padding.left + '" y1="' + yy + '" x2="' + (width - padding.right) + '" y2="' + yy + '" stroke="currentColor" opacity="0.2" stroke-dasharray="2,3" />');
      parts.push('<text x="' + (padding.left - 8) + '" y="' + yy + '" text-anchor="end" dominant-baseline="middle" font-size="11" fill="currentColor" opacity="0.6">' + compact(value) + "</text>");
    }

    var withYear = spanTo - spanFrom > 180 * dayMs;
    var xTicks = 6;
    for (var j = 0; j <= xTicks; j++) {
      var t = spanFrom + (spanTo - spanFrom) * j / xTicks;
      var anchor = j === 0 ? "start" : (j === xTicks ? "end" : "middle");
      parts.push('<text x="' + scaleX(t) + '" y="' + (height - padding.bottom + 16) + '" text-anchor="' + anchor + '" font-size="11" fill="currentColor" opacity="0.6">' + dateLabel(t, withYear) + "</text>");
    }

    var coords = points.map(function (point) {
      return scaleX(point.t) + "," + scaleY(point.total);
    });
    parts.push('<polyline points="' + coords.join(" ") + '" fill="none" stroke="' + themeAccent + '" stroke-width="2" />');

    if (points.length <= 60) {
      points.forEach(function (point) {
        parts.push('<circle cx="' + scaleX(point.t) + '" cy="' + scaleY(point.total) + '" r="2.5" fill="' + themeAccent + '" />');
      });
    }

    parts.push('<g class="hover-guide" style="display:none"><line class="hover-line" x1="0" y1="' + padding.top + '" x2="0" y2="' + (height - padding.bottom) + '" stroke="currentColor" opacity="0.5" stroke-dasharray="3,3" />' +
      '<circle class="hover-dot" r="4" fill="' + themeAccent + '" stroke="' + themeBg + '" stroke-width="1.5" /></g>');
    parts.push('<rect class="hover-capture" x="' + padding.left + '" y="' + padding.top + '" width="' + plotWidth + '" height="' + plotHeight + '" fill="transparent" />');

    svg.innerHTML = parts.join("");

    var capture = svg.querySelector(".hover-capture");
    capture.addEventListener("mousemove", function (event) { onHover(event, spanFrom, spanTo); });
    capture.addEventListener("mouseleave", onLeave);
  }

  function onHover(event, spanFrom, spanTo) {
    var svgPoint = svg.createSVGPoint();
    svgPoint.x = event.clientX;
    svgPoint.y = event.clientY;

    var ctm = svg.getScreenCTM();
    if (!ctm) {
      return;
    }

    var local = svgPoint.matrixTransform(ctm.inverse());
    var targetT = spanFrom + (local.x - padding.left) / plotWidth * (spanTo - spanFrom);

    var nearest = points[0];
    for (var i = 1; i < points.length; i++) {
      if (Math.abs(points[i].t - targetT) < Math.abs(nearest.t - targetT)) {
        nearest = points[i];
      }
    }

    var guide = svg.querySelector(".hover-guide");
    guide.style.display = "";
    var line = guide.querySelector(".hover-line");
    line.setAttribute("x1", scaleX(nearest.t));
    line.setAttribute("x2", scaleX(nearest.t));
    var dot = guide.querySelector(".hover-dot");
    dot.setAttribute("cx", scaleX(nearest.t));
    dot.setAttribute("cy", scaleY(nearest.total));

    tooltip.innerHTML = '<div class="budget-chart-tooltip-time">' + fullDateLabel(nearest.t) + " МСК</div>" +
      '<div class="budget-chart-tooltip-row">' + levelLabel(nearest) + "</div>" +
      '<div class="budget-chart-tooltip-row">💡 ' + grouped(nearest.total) + "</div>";
    tooltip.hidden = false;

    var canvasRect = canvas.getBoundingClientRect();
    var left = event.clientX - canvasRect.left + 12;
    var top = event.clientY - canvasRect.top - 10;
    if (left + tooltip.offsetWidth > canvasRect.width) {
      left = event.clientX - canvasRect.left - tooltip.offsetWidth - 12;
    }
    tooltip.style.left = left + "px";
    tooltip.style.top = top + "px";
  }

  function onLeave() {
    var guide = svg.querySelector(".hover-guide");
    if (guide) {
      guide.style.display = "none";
    }
    tooltip.hidden = true;
  }

  fetch(block.getAttribute("data-chart-url"), { credentials: "same-origin" })
    .then(function (response) { return response.json(); })
    .then(function (json) {
      if (json.error) {
        message(json.error, themeDanger);
        return;
      }

      from = new Date(json.from).getTime();
      to = new Date(json.to).getTime();
      points = (json.points || []).map(function (point) {
        return { t: new Date(point.at).getTime(), level: point.level, prestige: point.prestige, total: point.total };
      });
      render();
    })
    .catch(function () {
      message("не удалось загрузить график", themeDanger);
    });
})();
