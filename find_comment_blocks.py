import re, os

targets = []
for root, dirs, files in os.walk('backend'):
    dirs[:] = [d for d in dirs if d not in ('node_modules', 'dist', '.git')]
    for fn in files:
        if fn.endswith('.go'):
            targets.append(os.path.join(root, fn))

for path in targets:
    with open(path, encoding='utf-8', errors='replace') as f:
        lines = f.readlines()
    run_start = None
    run_lines = []
    for i, line in enumerate(lines):
        stripped = line.strip()
        is_comment_code_like = stripped.startswith('//') and len(stripped) > 2 and re.search(r'[;{}()=]|func |if |for |return ', stripped[2:])
        if is_comment_code_like:
            if run_start is None:
                run_start = i+1
            run_lines.append(stripped)
        else:
            if run_start is not None and len(run_lines) >= 5:
                print(f"{path}:{run_start}-{i} ({len(run_lines)} lines)")
                for rl in run_lines[:3]:
                    print("    " + rl)
                print("    ...")
            run_start = None
            run_lines = []
    if run_start is not None and len(run_lines) >= 5:
        print(f"{path}:{run_start}-{len(lines)} ({len(run_lines)} lines)")
