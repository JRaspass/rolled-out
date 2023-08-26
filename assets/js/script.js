// Navigation selects.
document.querySelector('body > select')?.addEventListener(
    'change', e => location = e.target.value);

// Sortable tables.
const table = document.querySelector('table.sort');
if (table) {
    const buttons = table.querySelectorAll('thead button');
    const tbody   = table.querySelector('tbody');

    // A numeric comparator.
    const cmp = (a, b, i, desc) => {
        a = a.children[i].dataset.sort;
        b = b.children[i].dataset.sort;

        // Blank cells always sort last.
        return (a == null) - (b == null) || (desc ? b - a : a - b);
    };

    for (const [i, btn] of buttons.entries())
        btn.addEventListener('click', () => {
            const desc = btn.className == 'asc';

            // Apply the new class.
            for (const btn of buttons) btn.className = '';
            btn.className = desc ? 'desc' : 'asc';

            // Sort the rows by the clicked column, tie-break with the first.
            tbody.replaceChildren(...[...tbody.children].sort(
                (a, b) => cmp(a, b, i, desc) || cmp(a, b, 0, false)));
        });
}

// Videos two-tier select.
const videoForm = document.querySelector('#videoForm');
if (videoForm) {
    (videoForm.stage.onchange = async () => {
        const url  = '/videos/' + videoForm.stage.value;
        const runs = await (await fetch(url)).json();

        videoForm.run.replaceChildren(...runs.map(r => {
            const option = document.createElement('option');
            option.innerText = (r.TimeRemaining / 1e9).toFixed(3) + ' ' + r.Player;
            option.value = r.Goal + '-' + r.TimeRemaining + '-' + r.Player;
            return option;
        }));
    })();
}
