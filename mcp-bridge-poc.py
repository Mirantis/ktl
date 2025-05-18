from mcp.server.fastmcp import FastMCP
from pathlib import Path
from subprocess import check_output

mcp = FastMCP("ktl")
CWD = Path(__file__).parent.absolute()
TOOL_DIR = CWD / "pkg" / "e2e" / "testdata"
TOOL_PREFIX = "mcp-"


@mcp.resource("reports://")
def reports() -> str:
    """
    Resource that lists all available reports.

    Use the 'describe' resource to obtain the description.
    Use the 'report' tool to obtain the contents.
    """

    names = [
        name.removeprefix(TOOL_PREFIX)
        for name in TOOL_DIR.glob(TOOL_PREFIX)
    ]

    return "\n".join([
        "Available reports:",
        "\n".join(f"- {name}" for name in names),
        "Use the 'report' tool to generate",
    ])


@mcp.resource("describe://{report}")
def describe(name: str) -> str:
    """
    Resource that returns the description of the report.

    Args:
      name: Name of the report
    """

    report_path = TOOL_DIR / f"{TOOL_PREFIX}{name}" / "pipeline.yaml"

    return check_output(
            ["go", "run", ".", "mcp", "describe", report_path],
            cwd=CWD, encoding='utf8',
    )

@mcp.tool()
def report(name: str) -> str:
    """
    Generate named K8s report and return the result in CSV format.

    Make sure to check the description via the 'describe' resource.

    Args:
      name: Name of the report
    """

    report_path = TOOL_DIR / f"{TOOL_PREFIX}{name}" / "pipeline.yaml"

    content =  check_output(
            ["go", "run", ".", "run", report_path],
            cwd=CWD, encoding='utf8',
    )

    return f"<source>{name}.csv</source>{content}"

