from mcp.server.fastmcp import FastMCP
from pathlib import Path
from subprocess import check_output

mcp = FastMCP("ktl")
CWD = Path(__file__).parent.absolute()
TOOL_DIR = CWD / "pkg" / "e2e" / "testdata"
TOOL_PREFIX = "mcp-"


@mcp.tool()
def list_reports() -> str:
    """
    Lists all available report names.

    Use the 'describe_report' tool to obtain the description.
    Use the 'report' tool to obtain the contents.
    """

    names = [
        tool_path.name.removeprefix(TOOL_PREFIX)
        for tool_path in TOOL_DIR.glob(f"{TOOL_PREFIX}*")
    ]

    return "\n".join([
        "Available reports:",
        "\n".join(f"- {name}" for name in names),
        "Use the 'report' tool to generate",
    ])


@mcp.tool()
def describe_report(name: str) -> str:
    """
    Show the description of the named report.

    Args:
      name: Name of the report
    """

    report_path = TOOL_DIR / f"{TOOL_PREFIX}{name}" / "pipeline.yaml"

    return check_output(
            ["go", "run", ".", "mcp", "describe", str(report_path)],
            cwd=CWD, encoding='utf8',
    )

@mcp.tool()
def report(name: str) -> str:
    """
    Generate named K8s report and return the result in CSV format.

    Make sure to check the list of available reports via the
    'list_reports' tool.

    Make sure to check the description via the 'describe_report' tool.

    Args:
      name: Name of the report
    """

    report_path = TOOL_DIR / f"{TOOL_PREFIX}{name}" / "pipeline.yaml"

    content =  check_output(
            ["go", "run", ".", "run", str(report_path)],
            cwd=CWD, encoding='utf8',
    )

    return f"<source>{name}.csv</source>{content}"

