import os
import re
import argparse
import subprocess
import requests
from pathlib import Path
from datetime import datetime
from typing import List, Tuple, Dict, Optional
from concurrent.futures import ThreadPoolExecutor, as_completed

# ========================
# Configurações
# ========================
GITHUB_TOKEN = os.environ.get("GITHUB_TOKEN")
GITHUB_USER = "seu-usuario"  # opcional, não é estritamente necessário
COMMIT_RANGE = "HEAD~10..HEAD"
BREAKING_PATTERNS = [r"BREAKING CHANGE", r"!\s", r"\bmajor\b"]

# Informe aqui seus repositórios locais
REPOSITORIES: Dict[str, str] = {
    # Control plane
    "control-plane_service-auth": "./.repos/control-plane_service-auth",
    "control-plane_job-manager": "./.repos/control-plane_job-manager",
    # Data plane
    "dataplane_runner": "./.repos/dataplane_runner",
    "dataplane_worker": "./.repos/dataplane_worker",
}


# ========================
# Utilitários
# ========================
def run_git(repo: Path, args: List[str]) -> str:
    return subprocess.run(
        ["git", "-C", str(repo)] + args,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        check=True,
    ).stdout.strip()


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


def get_commits(repo: Path) -> List[str]:
    try:
        log = run_git(repo, ["log", COMMIT_RANGE, "--pretty=format:%s"])
    except subprocess.CalledProcessError:
        # Fallback: get all commits if range fails
        try:
            log = run_git(repo, ["log", "--pretty=format:%s", "-10"])
        except subprocess.CalledProcessError:
            return []
    return [l for l in log.splitlines() if l.strip()]


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
        out.append("- Alinhamento de versão (sem mudanças de código neste serviço)")
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
    run_git(repo, ["commit", "-m", f"chore: align monolithic release to {version}"])
    run_git(repo, ["push", "--set-upstream", "origin", branch])


def create_tag(repo: Path, version: str):
    run_git(repo, ["tag", f"v{version}"])
    # Para enviar imediatamente a tag, descomente:
    # run_git(repo, ["push", "origin", f"v{version}"])


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
        print("❌ Não foi possível identificar repositório a partir do remote origin.")
        return

    body = (
        f"## Alinhamento de Release Monolítica v{version}\n\n"
        f"Este MR/PR atualiza a versão para `{version}` como parte do alinhamento monolítico.\n\n"
        f"**Tipo de bump:** {global_bump}\n\n"
        f"### Objetivo\n"
        f"Todos os serviços desta release usam a **mesma versão** para garantir compatibilidade entre componentes.\n\n"
        f"### Serviços atualizados nesta execução\n"
        + "\n".join(f"- {name}" for name in updated_repos) +
        f"\n\n### Checklist\n"
        f"- [x] Versão atualizada no arquivo VERSION\n"
        f"- [x] CHANGELOG.md atualizado\n"
        f"- [x] Tag criada localmente\n\n"
        f"**Nota:** Este é um alinhamento automático de versão. Revisar e aprovar."
    )

    if platform == "gitlab":
        print(f"🔁 Crie o merge request manualmente em:")
        print(
            f"   https://code.aws.dev/{owner}/{repo}/-/merge_requests/new?merge_request%5Bsource_branch%5D={branch}&merge_request%5Btarget_branch%5D=develop&merge_request%5Btitle%5D=Release%20v{version}%20%E2%80%94%20alinhamento%20monol%C3%ADtico"
        )
        return

    if platform == "github":
        if not GITHUB_TOKEN:
            print(f"🔁 Crie o pull request manualmente em:")
            print(f"   https://github.com/{owner}/{repo}/compare/develop...{branch}")
            return

        url = f"https://api.github.com/repos/{owner}/{repo}/pulls"
        headers = {"Authorization": f"token {GITHUB_TOKEN}"}
        data = {
            "title": f"Release v{version} — alinhamento monolítico",
            "head": branch,
            "base": "develop",
            "body": body,
        }
        response = requests.post(url, json=data, headers=headers)
        if response.ok:
            print(f"🔁 PR criado: {response.json().get('html_url')}")
        else:
            print(f"❌ Falha ao criar PR: {response.status_code} - {response.text}")


# ========================
# Fluxo principal
# ========================
def main():
    parser = argparse.ArgumentParser(
        description="Equaliza versões em múltiplos repositórios (monolithic alignment) com PR automático"
    )
    parser.add_argument(
        "--dry-run", action="store_true", help="Simula sem alterar nada"
    )
    args = parser.parse_args()

    if not GITHUB_TOKEN and not args.dry_run:
        print("⚠️  GITHUB_TOKEN não definido - PRs automáticos do GitHub serão manuais")
        print("   URLs para criação manual serão fornecidas quando necessário")

    # 1) Sincronizar repositórios para develop (sempre, mesmo em dry-run)
    print("\n🔄 Sincronizando repositórios com develop...")
    for name, path_str in REPOSITORIES.items():
        repo = Path(path_str)
        try:
            sync_repo_to_develop(repo)
            print(f"   ✅ {name} sincronizado")
        except Exception as e:
            print(f"   ⚠️  {name}: {str(e)}")
    
    # 2) Ler versões atuais e commits por repositório
    versions: Dict[str, str] = {}
    repo_commits: Dict[str, List[str]] = {}
    repo_bumps: Dict[str, str] = {}

    for name, path_str in REPOSITORIES.items():
        repo = Path(path_str)
        print(f"\n🔍 Inspecionando {name} ({repo})")

        version_path = repo / "VERSION"
        current_version = (
            version_path.read_text().strip() if version_path.exists() else "0.0.0"
        )
        versions[name] = current_version

        commits = get_commits(repo)
        repo_commits[name] = commits

        bump_type, _ = detect_bump_type(commits)
        repo_bumps[name] = bump_type

        print(f"   • Versão atual: {current_version}")
        print(f"   • Commits (últimos 10): {len(commits)}")
        print(f"   • Bump sugerido por este repo: {bump_type}")

    # 3) Calcular bump GLOBAL (prioridade major > minor > patch)
    if any(b == "major" for b in repo_bumps.values()):
        global_bump = "major"
    elif any(b == "minor" for b in repo_bumps.values()):
        global_bump = "minor"
    else:
        global_bump = "patch"

    # 4) Escolher versão-base GLOBAL = maior semver entre todos
    global_base = "0.0.0"
    for v in versions.values():
        if semver_gt(v, global_base):
            global_base = v

    new_version = bump_version(global_base, global_bump)

    print("\n================ Gerindo como monolito ================")
    print(f"Versão-base global: {global_base}")
    print(f"Bump global: {global_bump}")
    print(f"➡️  Nova versão global: {new_version}")
    print("======================================================")

    # 5) Aplicar nova versão a todos os repositórios
    def process_repo(name: str) -> Tuple[Optional[str], str]:
        repo = Path(REPOSITORIES[name])
        current_version = versions[name]
        commits = repo_commits[name]
        _, relevant = detect_bump_type(commits)
        aligned_only = len(relevant) == 0
        branch_name = f"atualizacao-versao-v{new_version}"
        
        log_lines = []
        log_lines.append(f"📦 {name}")
        log_lines.append(f"   • Versão atual: {current_version}")
        log_lines.append(f"   • Nova versão global: {new_version}")
        log_lines.append(f"   • Branch: {branch_name}")

        if args.dry_run:
            if current_version == new_version:
                log_lines.append("   💡 [simulação] Já está alinhado. Nenhuma ação seria necessária.")
                return None, "\n".join(log_lines)
            log_lines.append(f"   💡 [simulação] Processaria: {branch_name}")
            return name, "\n".join(log_lines)

        if current_version != new_version:
            try:
                create_branch(repo, branch_name)
                update_version_file(repo, new_version)
                update_changelog(repo, new_version, relevant, aligned_only=aligned_only)
                commit_and_push(repo, branch_name, new_version)
                create_tag(repo, new_version)
                log_lines.append(f"   ✅ {name} processado")
                return name, "\n".join(log_lines)
            except Exception as e:
                log_lines.append(f"   ❌ Erro ao processar {name}: {str(e)}")
                return None, "\n".join(log_lines)
        else:
            log_lines.append("   ✔️ Já está alinhado. Pulando alterações.")
            return None, "\n".join(log_lines)

    # Process repos in parallel and collect results
    updated_repos: List[str] = []
    repo_logs: List[str] = []
    
    with ThreadPoolExecutor(max_workers=4) as executor:
        futures = {executor.submit(process_repo, name): name for name in REPOSITORIES.keys()}
        for future in as_completed(futures):
            result, log = future.result()
            repo_logs.append(log)
            if result:
                updated_repos.append(result)
    
    # Print all logs in order
    print("\n================ Processamento dos Repositórios ================")
    for log in sorted(repo_logs):
        print(log)
    print("================================================================")

    # 6) Abrir PRs (um por repositório alterado)
    if args.dry_run:
        if updated_repos:
            print("\n📝 [simulação] PRs/MRs seriam criados para os repositórios:")
            for r in updated_repos:
                print(f"   - {r} (base: develop, head: atualizacao-versao-v{new_version})")
        else:
            print("\n📝 [simulação] Tudo já estava alinhado. Nenhum PR/MR seria necessário.")
    else:
        if updated_repos:
            print("\n🔁 Criando PRs/MRs...")
            def create_pr_for_repo(name):
                repo_path = Path(REPOSITORIES[name])
                owner, repo, platform = get_repo_info(repo_path)
                if platform == "gitlab":
                    return f"🔁 {name}: https://code.aws.dev/{owner}/{repo}/-/merge_requests/new?merge_request%5Bsource_branch%5D=atualizacao-versao-v{new_version}&merge_request%5Btarget_branch%5D=develop&merge_request%5Btitle%5D=Release%20v{new_version}%20%E2%80%94%20alinhamento%20monol%C3%ADtico"
                return f"🔁 {name}: GitHub PR criado"

            pr_results = []
            with ThreadPoolExecutor(max_workers=4) as executor:
                futures = [executor.submit(create_pr_for_repo, name) for name in updated_repos]
                for future in as_completed(futures):
                    pr_results.append(future.result())
            
            print("\n================ Links dos Merge Requests ================")
            for result in sorted(pr_results):
                print(result)
            print("==========================================================")

    print("\n✅ Concluído.")


if __name__ == "__main__":
    main()
