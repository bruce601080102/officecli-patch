"""Allow ``python -m officecli_patch`` as a PATH-independent fallback."""

from .launcher import main


if __name__ == "__main__":
    main()
