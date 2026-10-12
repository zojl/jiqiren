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
