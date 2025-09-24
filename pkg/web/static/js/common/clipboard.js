/*
This function activates copy and paste buttons that are on text inputs.
*/
export function activateCopyButtons() {
  const btns = document.querySelectorAll('[data-clipboard-target]');
  btns.forEach(btn => {
    btn.addEventListener('click', function() {
      const target = btn.dataset.clipboardTarget;
      const value = document.querySelector(target).value;
      navigator.clipboard.writeText(value)
        .then(() => {
          btn.classList.add('btn-success');
          btn.classList.remove('btn-outline-secondary');
          btn.innerHTML = '<i class="fas fa-fw fa-copy"></i>';
        })
        .catch(() => {
          btn.classList.add('btn-danger');
          btn.classList.remove('btn-outline-secondary');
          btn.innerHTML = '<i class="fas fa-fw fa-times"></i>';
        })
        .finally(() => {
          setTimeout(() => {
            btn.classList.remove('btn-success', 'btn-danger');
            btn.classList.add('btn-outline-secondary');
            btn.innerHTML = '<i class="fas fa-fw fa-copy"></i>';
          }, 500);
        });
    });
  });
}