// Memastikan script jalan setelah HTML siap
document.addEventListener('DOMContentLoaded', () => {

    function fmt(bytes, d = 1) {
        if (!bytes) return '0 B';
        const k = 1024, sz = ['B','KB','MB','GB','TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return (bytes / Math.pow(k, i)).toFixed(d) + ' ' + sz[i];
    }

    function setWidth(id, pct) {
        const el = document.getElementById(id);
        if (el) el.style.width = Math.min(100, Math.max(0, pct)) + '%';
    }

    function setText(id, val) {
        const el = document.getElementById(id);
        if (el) el.textContent = val;
    }

    function setBigVal(id, pct) {
        const el = document.getElementById(id);
        if (!el) return;
        el.innerHTML = `${parseFloat(pct).toFixed(1)}<span style="font-size:0.45em;opacity:0.6">%</span>`;
    }

    function setBarColor(id, pct) {
        const el = document.getElementById(id);
        if (!el) return;
        if (pct > 85) {
            el.style.background = 'var(--warn)';
            el.style.boxShadow = '0 0 8px var(--warn)';
        } else {
            el.style.background = '';
            el.style.boxShadow = '';
        }
    }

    function updateTemp(temp) {
        const tempEl   = document.getElementById('temp-value');
        const statusEl = document.getElementById('temp-status');
        const fillEl   = document.getElementById('thermo-fill');
        const bulbEl   = document.getElementById('thermo-bulb');

        if (!temp || temp <= 0) {
            if (tempEl) tempEl.innerHTML = 'N/A<span class="unit">°C</span>';
            if (statusEl) { statusEl.textContent = 'NOT AVAILABLE'; statusEl.style.color = 'var(--text-muted)'; }
            return;
        }

        const ratio  = Math.min(1, Math.max(0, (temp - 20) / 80)); 
        const fillH  = 4 + ratio * 58;   
        const fillY  = 66 - fillH;        

        let color, status;
        if (temp < 50)      { color = '#00ff9d'; status = 'COOL'; }
        else if (temp < 70) { color = '#ff9f43'; status = 'WARM'; }
        else if (temp < 85) { color = '#ff6b35'; status = 'HOT'; }
        else                { color = '#ff3838'; status = '⚠ CRITICAL'; }

        if (tempEl)  {
            tempEl.innerHTML = `${temp.toFixed(1)}<span class="unit">°C</span>`;
            tempEl.style.color  = color;
        }
        if (statusEl){ statusEl.textContent = status; statusEl.style.color = color; }
        if (fillEl)  { fillEl.setAttribute('y', fillY); fillEl.setAttribute('height', fillH); fillEl.style.fill = color; }
        if (bulbEl)  bulbEl.style.fill = color;
    }

    async function fetchMonitorData() {
        try {
            const response = await fetch('/api/monitor'); // Pastikan API ini memang ada di servermu
            const data = await response.json();

            const cpuPct  = parseFloat(data.cpu.percent).toFixed(1);
            const memPct  = parseFloat(data.memory.used_percent).toFixed(1);
            const diskPct = parseFloat(data.disk.used_percent).toFixed(1);

            setText('hostname', data.hostname || 'Unknown');
            setText('os', `${data.os} ${data.platform}`);
            setText('timestamp', new Date(data.last_update).toLocaleTimeString('id-ID'));

            setBigVal('cpu-percent', cpuPct);
            setWidth('cpu-fill', cpuPct);
            setBarColor('cpu-fill', cpuPct);
            setText('cpu-cores', data.cpu.cores + ' cores');
            setText('cpu-load', cpuPct + '%');
            setText('cpu-model', (data.cpu.model || '').substring(0, 36));

            setBigVal('mem-percent', memPct);
            setWidth('mem-fill', memPct);
            setBarColor('mem-fill', memPct);
            setText('mem-used', fmt(data.memory.used));
            setText('mem-total', fmt(data.memory.total));
            setText('mem-free', fmt(data.memory.free));
            setText('mem-cached', fmt(data.memory.cached || 0));

            setBigVal('disk-percent', diskPct);
            setWidth('disk-fill', diskPct);
            setBarColor('disk-fill', diskPct);
            setText('disk-used', fmt(data.disk.used));
            setText('disk-total', fmt(data.disk.total));
            setText('disk-free', fmt(data.disk.free));

            updateTemp(data.temperature || 0);
            setText('proc-count', data.processes ?? '—');
            if (data.load_average) {
                setText('load-1',  (data.load_average.load1  || 0).toFixed(2));
                setText('load-5',  (data.load_average.load5  || 0).toFixed(2));
                setText('load-15', (data.load_average.load15 || 0).toFixed(2));
            }

            if (data.network) {
                setText('net-sent', fmt(data.network.bytes_sent));
                setText('net-recv', fmt(data.network.bytes_recv));
                setText('net-conn', data.network.connections ?? '—');
            }

            setText('info-hostname', data.hostname || '—');
            setText('info-os', data.os || '—');
            setText('info-platform', data.platform || '—');
            setText('info-version', data.platform_version || '—');

            const s = data.uptime || 0;
            const d = Math.floor(s / 86400);
            const h = Math.floor((s % 86400) / 3600);
            const m = Math.floor((s % 3600) / 60);
            setText('uptime-display', `${d}d ${h}h ${m}m`);
            setText('uptime-d', d);
            setText('uptime-h', h);
            setText('uptime-m', m);

        } catch (err) {
            console.error('Fetch error:', err);
            setText('timestamp', 'ERR');
        }
    }

    // Jalankan pertama kali
    fetchMonitorData();
    // Ulangi setiap 2 detik
    setInterval(fetchMonitorData, 2000);
});