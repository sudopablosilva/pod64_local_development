import os
import re
import argparse
import subprocess
import requests
from pathlib import Path
from datetime import datetime
from typing import List, Tuple, Dict, Optional
from concurrent.futures import ThreadPoolExecutor, as_completed
import urllib3

# Disable SSL warnings for corporate environments
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

# ========================
# Configuracoes
# ========================
GITHUB_TOKEN = os.environ.get("GITHUB_TOKEN")
GITHUB_USER = "seu-usuario"  # opcional, nao e estritamente necessario
COMMIT_RANGE = "HEAD~10..HEAD"
BREAKING_PATTERNS = [r"BREAKING CHANGE", r"!\s", r"\bmajor\b"]

# Repositorios para teste local
TEST_REPOSITORIES: Dict[str, str] = {
    # Data plane
    "dataplane_runner": "./.repos/dataplane_runner",
    "dataplane_worker": "./.repos/dataplane_worker",
    # Control plane
    "control-plane_service-auth": "./.repos/control-plane_service-auth",
    "control-plane_job-manager": "./.repos/control-plane_job-manager",
}

# Repositorios para ambiente Itau
ITAU_REPOSITORIES: Dict[str, str] = {
    # Control plane
    "control-plane_itau-ns7-container-job-manager-runner": "./.repos/itau-ns7-container-job-manager-runner",
    "control-plane_itau-ns7-container-job-manager-worker": "./.repos/itau-ns7-container-job-manager-worker",
    # Data plane
    "dataplane_itau-ns7-container-scheduler-manager": "./.repos/itau-ns7-container-scheduler-manager",
    "dataplane_itau-ns7-container-scheduler-adapter": "./.repos/itau-ns7-container-scheduler-adapter",
}


# ========================
# Utilitarios
# ========================
def run_git(repo: Path, args: List[str]) -> str:
    result = subprocess.run(
        ["git", "-C", str(repo)] + args,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    if result.returncode != 0:
        raise subprocess.CalledProcessError(
            result.returncode, result.args, output=result.stdout, stderr=result.stderr
        )
    return result.stdout.strip()


def sync_repo_to_develop(repo: Path):
    """Ensure repo is on develop branch and up to date"""
    try:
        run_git(repo, ["checkout", "develop"])
        run_git(repo, ["pull", "origin", "develop"])
    except subprocess.CalledProcessError:
        # Create develop from main/master if it doesn't exist
        try:
            run_git(repo, ["checkout", "main"])
            run_git(repo, ["pull", "origin", "main"])
            run_git(repo, ["checkout", "-b", "develop"])
            run_git(repo, ["push", "-u", "origin", "develop"])
        except subprocess.CalledProcessError:
            run_git(repo, ["checkout", "master"])
            run_git(repo, ["pull", "origin", "master"])
            run_git(repo, ["checkout", "-b", "develop"])
            run_git(repo, ["push", "-u", "origin", "develop"])


def get_commits_since_last_tag(repo: Path) -> List[str]:
    """Get commits since last version tag, avoiding reprocessing"""
    try:
        # Try to get last version tag
        last_tag = run_git(repo, ["describe", "--tags", "--abbrev=0", "--match=v*"])
        # Get commits since last tag
        log = run_git(repo, ["log", f"{last_tag}..HEAD", "--pretty=format:%s"])
    except subprocess.CalledProcessError:
        # No tags found, get recent commits
        try:
            log = run_git(repo, ["log", "--pretty=format:%s", "-10"])
        except subprocess.CalledProcessError:
            return []
    return [l for l in log.splitlines() if l.strip()]


def get_commits(repo: Path) -> List[str]:
    """Get commits for analysis - uses tag-based approach to avoid reprocessing"""
    return get_commits_since_last_tag(repo)


def detect_bump_type(commits: List[str]) -> Tuple[str, List[str]]:
    bump = "patch"
    relevant = []
    for commit in commits:
        if any(re.search(p, commit, re.IGNORECASE) for p in BREAKING_PATTERNS):
            return "major", commits
        elif "feat" in commit.lower():
            bump = "minor"
            relevant.append(commit)
        elif "fix" in commit.lower():
            relevant.append(commit)
    return bump, relevant or commits


def parse_semver(v: str) -> Tuple[int, int, int]:
    major, minor, patch = map(int, v.strip().split("."))
    return major, minor, patch


def semver_gt(a: str, b: str) -> bool:
    return parse_semver(a) > parse_semver(b)


def bump_version(version: str, bump_type: str) -> str:
    major, minor, patch = parse_semver(version)
    if bump_type == "major":
        return f"{major + 1}.0.0"
    elif bump_type == "minor":
        return f"{major}.{minor + 1}.0"
    return f"{major}.{minor}.{patch + 1}"


def update_version_file(repo: Path, new_version: str):
    (repo / "VERSION").write_text(new_version + "\n")


def format_changelog(
    version: str, commits: List[str], aligned_only: bool = False
) -> str:
    date = datetime.today().strftime("%Y-%m-%d")
    out = [f"## [{version}] - {date}"]
    if aligned_only:
        out.append("- Alinhamento de versao (sem mudancas de codigo neste servico)")
    else:
        for commit in commits:
            out.append(f"- {commit}")
    return "\n".join(out) + "\n\n"


def update_changelog(repo: Path, version: str, commits: List[str], aligned_only: bool):
    changelog_path = repo / "CHANGELOG.md"
    existing = changelog_path.read_text() if changelog_path.exists() else ""
    new_entry = format_changelog(version, commits, aligned_only=aligned_only)
    changelog_path.write_text(new_entry + existing)


def create_branch(repo: Path, branch: str):
    # Delete branch if it already exists
    try:
        run_git(repo, ["branch", "-D", branch])
    except subprocess.CalledProcessError:
        pass  # Branch doesn't exist, continue

    # Repo should already be on develop and up to date
    run_git(repo, ["checkout", "-b", branch])


def commit_and_push(repo: Path, branch: str, version: str):
    run_git(repo, ["add", "VERSION", "CHANGELOG.md"])
    run_git(repo, ["commit", "-m", f"chore: alinha release monolitico para {version}"])
    # Force push to handle non-fast-forward issues
    run_git(repo, ["push", "--force-with-lease", "--set-upstream", "origin", branch])


def create_tag(repo: Path, version: str):
    run_git(repo, ["tag", f"v{version}"])
    # Push tag immediately to mark commits as processed
    run_git(repo, ["push", "origin", f"v{version}"])


def get_repo_info(repo_path: Path) -> Tuple[str, str, str]:
    try:
        url = run_git(repo_path, ["remote", "get-url", "origin"])

        # GitHub itau-corp
        github_parts = re.findall(r"github.com[:/]itau-corp/(.+?)(\.git)?$", url)
        if github_parts:
            return "itau-corp", github_parts[0][0], "github"

        # AWS CodeCommit (GitLab)
        aws_parts = re.findall(r"code\.aws\.dev[:/](.+)/(.+?)(\.git)?$", url)
        if aws_parts:
            return aws_parts[0][0], aws_parts[0][1], "gitlab"

        return "", "", ""
    except subprocess.CalledProcessError:
        return "", "", ""


def create_pull_request(
    repo_path: Path,
    branch: str,
    version: str,
    global_bump: str,
    updated_repos: List[str],
):
    owner, repo, platform = get_repo_info(repo_path)
    if not owner or not repo:
        print(
            "[ERROR] Nao foi possivel identificar repositorio a partir do remote origin."
        )
        return

    body = (
        f"## Alinhamento de Release Monolitica v{version}\n\n"
        f"Este MR/PR atualiza a versao para `{version}` como parte do alinhamento monolitico.\n\n"
        f"**Tipo de bump:** {global_bump}\n\n"
        f"### Objetivo\n"
        f"Todos os servicos desta release usam a **mesma versao** para garantir compatibilidade entre componentes.\n\n"
        f"### Servicos atualizados nesta execucao\n"
        + "\n".join(f"- {name}" for name in updated_repos)
        + f"\n\n### Checklist\n"
        f"- [x] Versao atualizada no arquivo VERSION\n"
        f"- [x] CHANGELOG.md atualizado\n"
        f"- [x] Tag criada localmente\n\n"
        f"**Nota:** Este e um alinhamento automatico de versao. Revisar e aprovar."
    )

    if platform == "gitlab":
        print(f"[MR] Crie o merge request manualmente em:")
        print(
            f"   https://code.aws.dev/{owner}/{repo}/-/merge_requests/new?merge_request%5Bsource_branch%5D={branch}&merge_request%5Btarget_branch%5D=develop&merge_request%5Btitle%5D=Release%20v{version}%20%E2%80%94%20alinhamento%20monolitico"
        )
        return

    if platform == "github":
        if not GITHUB_TOKEN:
            print(f"[PR] Crie o pull request manualmente em:")
            print(f"   https://github.com/{owner}/{repo}/compare/develop...{branch}")
            return

        url = f"https://api.github.com/repos/{owner}/{repo}/pulls"
        headers = {"Authorization": f"token {GITHUB_TOKEN}"}
        data = {
            "title": f"Release v{version} - alinhamento monolitico",
            "head": branch,
            "base": "develop",
            "body": body,
        }
        try:
            response = requests.post(
                url, json=data, headers=headers, verify=False, timeout=10
            )
            if response.ok:
                print(f"[PR] PR criado: {response.json().get('html_url')}")
            else:
                print(
                    f"[ERROR] Falha ao criar PR: {response.status_code} - {response.text}"
                )
                print(f"[PR] Crie o pull request manualmente em:")
                print(
                    f"   https://github.com/{owner}/{repo}/compare/develop...{branch}"
                )
        except Exception as e:
            print(f"[ERROR] Erro SSL/Rede: {str(e)}")
            print(f"[PR] Crie o pull request manualmente em:")
            print(f"   https://github.com/{owner}/{repo}/compare/develop...{branch}")


# ========================
# Fluxo principal
# ========================
def main():
    parser = argparse.ArgumentParser(
        description="Equaliza versoes em multiplos repositorios (monolithic alignment) com PR automatico"
    )
    parser.add_argument(
        "--dry-run", action="store_true", help="Simula sem alterar nada"
    )
    parser.add_argument(
        "--itau", action="store_true", help="Usa repositorios do ambiente Itau"
    )
    args = parser.parse_args()

    # Select repositories based on flag
    REPOSITORIES = ITAU_REPOSITORIES if args.itau else TEST_REPOSITORIES

    if args.itau:
        print("[INFO] Usando repositorios do ambiente Itau")
    else:
        print("[INFO] Usando repositorios de teste local (AWS)")

    if not GITHUB_TOKEN and not args.dry_run:
        print(
            "[WARN] GITHUB_TOKEN nao definido - PRs automaticos do GitHub serao manuais"
        )
        print("   URLs para criacao manual serao fornecidas quando necessario")

    # 1) Sincronizar repositorios para develop (sempre, mesmo em dry-run)
    print("\n[SYNC] Sincronizando repositorios com develop...")
    for name, path_str in REPOSITORIES.items():
        repo = Path(path_str)
        try:
            sync_repo_to_develop(repo)
            print(f"   [OK] {name} sincronizado")
        except Exception as e:
            print(f"   [WARN] {name}: {str(e)}")

    # 2) Ler versoes atuais e commits por repositorio
    versions: Dict[str, str] = {}
    repo_commits: Dict[str, List[str]] = {}
    repo_bumps: Dict[str, str] = {}

    for name, path_str in REPOSITORIES.items():
        repo = Path(path_str)
        print(f"\n[INFO] Inspecionando {name} ({repo})")

        version_path = repo / "VERSION"
        current_version = (
            version_path.read_text().strip() if version_path.exists() else "0.0.0"
        )
        versions[name] = current_version

        commits = get_commits(repo)
        repo_commits[name] = commits

        bump_type, _ = detect_bump_type(commits)
        repo_bumps[name] = bump_type

        # Debug: show which commits are being considered
        if commits:
            print(
                f"   - Commits encontrados: {commits[:3]}{'...' if len(commits) > 3 else ''}"
            )
        else:
            print(f"   - Nenhum commit novo desde ultima tag")

        print(f"   - Versao atual: {current_version}")
        print(f"   - Commits desde ultima tag: {len(commits)}")
        print(f"   - Bump sugerido por este repo: {bump_type}")

    # 3) Calcular bump GLOBAL (prioridade major > minor > patch)
    if any(b == "major" for b in repo_bumps.values()):
        global_bump = "major"
    elif any(b == "minor" for b in repo_bumps.values()):
        global_bump = "minor"
    else:
        global_bump = "patch"

    # 4) Escolher versao-base GLOBAL = maior semver entre todos
    global_base = "0.0.0"
    for v in versions.values():
        if semver_gt(v, global_base):
            global_base = v

    new_version = bump_version(global_base, global_bump)

    print("\n================ Gerindo como monolito ================")
    print(f"Versao-base global: {global_base}")
    print(f"Bump global: {global_bump}")
    print(f"=> Nova versao global: {new_version}")
    print("======================================================")

    # 5) Aplicar nova versao a todos os repositorios
    def process_repo(name: str) -> Tuple[Optional[str], str]:
        repo = Path(REPOSITORIES[name])
        current_version = versions[name]
        commits = repo_commits[name]
        _, relevant = detect_bump_type(commits)
        aligned_only = len(relevant) == 0
        branch_name = f"atualizacao-versao-v{new_version}"

        log_lines = []
        log_lines.append(f"[REPO] {name}")
        log_lines.append(f"   - Versao atual: {current_version}")
        log_lines.append(f"   - Nova versao global: {new_version}")
        log_lines.append(f"   - Branch: {branch_name}")

        if args.dry_run:
            if current_version == new_version:
                log_lines.append(
                    "   [SIM] Ja esta alinhado. Nenhuma acao seria necessaria."
                )
                return None, "\n".join(log_lines)
            log_lines.append(f"   [SIM] Processaria: {branch_name}")
            return name, "\n".join(log_lines)

        if current_version != new_version:
            try:
                create_branch(repo, branch_name)
                update_version_file(repo, new_version)
                update_changelog(repo, new_version, relevant, aligned_only=aligned_only)
                commit_and_push(repo, branch_name, new_version)
                create_tag(repo, new_version)
                log_lines.append(f"   [OK] {name} processado")
                return name, "\n".join(log_lines)
            except subprocess.CalledProcessError as e:
                error_msg = e.stderr.strip() if e.stderr else str(e)
                log_lines.append(f"   [ERROR] Git erro: {error_msg}")
                return None, "\n".join(log_lines)
            except Exception as e:
                log_lines.append(f"   [ERROR] Erro ao processar {name}: {str(e)}")
                return None, "\n".join(log_lines)
        else:
            log_lines.append("   [SKIP] Ja esta alinhado. Pulando alteracoes.")
            return None, "\n".join(log_lines)

    # Process repos in parallel and collect results
    updated_repos: List[str] = []
    repo_logs: List[str] = []

    with ThreadPoolExecutor(max_workers=4) as executor:
        futures = {
            executor.submit(process_repo, name): name for name in REPOSITORIES.keys()
        }
        for future in as_completed(futures):
            result, log = future.result()
            repo_logs.append(log)
            if result:
                updated_repos.append(result)

    # Print all logs in order
    print("\n================ Processamento dos Repositorios ================")
    for log in sorted(repo_logs):
        print(log)
    print("================================================================")

    # 6) Abrir PRs (um por repositorio alterado)
    if args.dry_run:
        if updated_repos:
            print("\n[SIM] PRs/MRs seriam criados para os repositorios:")
            for r in updated_repos:
                print(
                    f"   - {r} (base: develop, head: atualizacao-versao-v{new_version})"
                )
        else:
            print("\n[SIM] Tudo ja estava alinhado. Nenhum PR/MR seria necessario.")
    else:
        if updated_repos:
            print("\n[PR] Criando PRs/MRs...")

            def create_pr_for_repo(name):
                repo_path = Path(REPOSITORIES[name])
                create_pull_request(
                    repo_path,
                    f"atualizacao-versao-v{new_version}",
                    new_version,
                    global_bump,
                    updated_repos,
                )
                return f"[DONE] {name}: PR/MR processado"

            pr_results = []
            with ThreadPoolExecutor(max_workers=4) as executor:
                futures = [
                    executor.submit(create_pr_for_repo, name) for name in updated_repos
                ]
                for future in as_completed(futures):
                    pr_results.append(future.result())

            print("\n================ Resultado dos PRs/MRs ================")
            for result in sorted(pr_results):
                print(result)
            print("======================================================")

    print("\n[DONE] Concluido.")


if __name__ == "__main__":
    main()
