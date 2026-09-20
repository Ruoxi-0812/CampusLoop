#!/usr/bin/env python3
"""Build baseline/current from identical test harness and run local gRPC comparisons."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import statistics
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[2]
SERVICE = ROOT / 'src/productcatalogservice'
OUT = Path(__file__).resolve().parent
BASE = '069c0c1bb3dd7b7adc2de005a323af1b218078df'

def run(args, **kwargs):
    return subprocess.run(args, check=True, **kwargs)

with tempfile.TemporaryDirectory(prefix='campusloop-compare-') as temp:
    temp = Path(temp)
    baseline = temp / 'baseline'
    shutil.copytree(SERVICE, baseline)
    for filename in ['product_catalog.go', 'server.go']:
        original = subprocess.check_output(['git', 'show', f'{BASE}:src/productcatalogservice/{filename}'], cwd=ROOT)
        (baseline / filename).write_bytes(original)
    harness = (baseline / 'performance_test.go').read_text().replace('reloadCatalog.Store(false)', 'reloadCatalog = false')
    (baseline / 'performance_test.go').write_text(harness)
    # Added correctness tests target the new snapshot API and are not baseline code.
    (baseline / 'catalog_index_test.go').unlink(missing_ok=True)
    binaries = {}
    for variant, source in [('before', baseline), ('after', SERVICE)]:
        binary = temp / f'{variant}.test'
        run(['go', 'test', '-c', '-o', str(binary), '.'], cwd=source)
        binaries[variant] = binary
    metadata = {'baseline_commit': BASE, 'sha256': {name: hashlib.sha256((SERVICE/name).read_bytes()).hexdigest() for name in ['product_catalog.go','server.go','performance_test.go','products.json']}, 'order': []}
    for size in [9, 1000, 10000]:
        order = ['after', 'before'] if size == 1000 else ['before', 'after']
        for variant in order:
            label = f'{variant}-{size}'
            metadata['order'].append(label)
            env = dict(os.environ, CATALOG_PERF_OUTPUT=str(OUT/f'{label}.json'), CATALOG_PERF_SIZE=str(size), CATALOG_PERF_LOOKUP_ONLY='1', CATALOG_PERF_VARIANT=variant)
            print(f'Running {label}', flush=True)
            completed = run([str(binaries[variant]), '-test.run=^TestPerformanceBaseline$', '-test.v', '-test.timeout=2m'], cwd=SERVICE, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
            (OUT/f'{label}.log').write_text(completed.stdout)
            print(completed.stdout, flush=True)
    (OUT/'comparison-metadata.json').write_text(json.dumps(metadata, indent=2)+'\n')

lines = ['# Product lookup optimization results', '', 'Local loopback gRPC; 20 concurrent workers; synthetic catalogs cloned from actual product fields. Three 5-second measured repetitions per variant/size, 100 warmup requests per repetition. Values below are medians of per-run values.', '', '| Products | Before p95 ms | After p95 ms | p95 reduction | Before requests/s | After requests/s | Throughput increase |', '|---:|---:|---:|---:|---:|---:|---:|']
for size in [9,1000,10000]:
    rows = [json.loads((OUT/f'{v}-{size}.json').read_text())['results'] for v in ['before','after']]
    before, after = [statistics.median(r['p95_ms'] for r in group) for group in rows]
    brps, arps = [statistics.median(r['rps'] for r in group) for group in rows]
    lines.append(f'| {size:,} | {before:.6f} | {after:.6f} | {(before-after)/before*100:.2f}% | {brps:,.0f} | {arps:,.0f} | {(arps-brps)/brps*100:.2f}% |')
lines += ['', 'Reductions are (before − after) / before; throughput increases are (after − before) / before. Negative values mean regression. Raw per-run results and execution order are saved alongside this report.', '', 'Scope: single process hosting both client and server, one shared HTTP/2 connection, in-memory catalog, no tracing interceptors, database, frontend, Kubernetes, or external network. Each request validates the returned product. Index construction is included in warmup rather than steady-state latency. This tests GetProduct only and does not establish whole-site speed or production capacity. Each variant is run as a block of three repetitions; time/order effects and local scheduling remain possible.']
(OUT/'COMPARISON.md').write_text('\n'.join(lines)+'\n')
print('\n'.join(lines))
