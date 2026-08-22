/**
 * CSV inventory export of the visible topology (nodes + adjacency).
 * Useful for spreadsheets / MSP docs. Lucidchart network diagrams import
 * draw.io files — use the Lucid / Draw.io buttons for that.
 */
'use strict';

const TopologyCSVExport = {

  _escape(cell) {
    const s = String(cell == null ? '' : cell);
    if (/[",\n\r]/.test(s)) return '"' + s.replace(/"/g, '""') + '"';
    return s;
  },

  _row(cells) {
    return cells.map(c => this._escape(c)).join(',');
  },

  /**
   * @returns {{ nodes: number, edges: number, csv: string }}
   */
  generate(viewer) {
    if (!viewer || !viewer.cy || !viewer.rawData) {
      throw new Error('no map loaded');
    }
    const visible = new Set();
    viewer.cy.nodes(':visible').forEach(n => visible.add(n.id()));
    if (visible.size === 0) throw new Error('nothing visible to export');

    const lines = [];
    lines.push(this._row(['type', 'name', 'ip', 'platform', 'discovered', 'peer', 'local_if', 'remote_if']));

    let edges = 0;
    viewer.cy.nodes(':visible').forEach(n => {
      const d = n.data();
      lines.push(this._row([
        'node',
        d.id || d.label || '',
        d.ip || '',
        d.platform || '',
        d.discovered ? 'yes' : 'no',
        '', '', '',
      ]));
    });

    const raw = viewer.rawData;
    for (const [name, device] of Object.entries(raw)) {
      if (!visible.has(name)) continue;
      const peers = (device && device.peers) || {};
      for (const [peer, entry] of Object.entries(peers)) {
        if (!visible.has(peer) && !visible.has(name)) continue;
        const conns = (entry && entry.connections) || [];
        if (conns.length === 0) {
          lines.push(this._row(['link', name, '', '', '', peer, '', '']));
          edges++;
          continue;
        }
        for (const c of conns) {
          const localIf = Array.isArray(c) ? (c[0] || '') : '';
          const remoteIf = Array.isArray(c) ? (c[1] || '') : '';
          lines.push(this._row(['link', name, '', '', '', peer, localIf, remoteIf]));
          edges++;
        }
      }
    }

    return { nodes: visible.size, edges: edges, csv: lines.join('\n') + '\n' };
  },

  download(viewer, mapName) {
    const title = (mapName || 'map').replace(/\.json$/i, '');
    const out = this.generate(viewer);
    const blob = new Blob([out.csv], { type: 'text/csv;charset=utf-8' });
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = title + '-topology.csv';
    a.click();
    URL.revokeObjectURL(a.href);
    return out;
  },
};

if (typeof window !== 'undefined') window.TopologyCSVExport = TopologyCSVExport;
if (typeof module !== 'undefined' && module.exports) module.exports = { TopologyCSVExport };
