import { OperionLogo } from "@/components/icons/OperionLogo";

export function Header() {
  return (
    <header className="sticky top-0 z-50 w-full border-b border-border/40 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container flex h-16 max-w-screen-2xl items-center justify-between">
        <a
          href="/"
          className="flex items-center space-x-2"
          aria-label="Operion Flow Home"
        >
          <OperionLogo className="h-6 w-auto" />
          <span className="font-bold text-lg hidden sm:inline-block">
            Editor
          </span>
        </a>
        {/* <nav className="flex items-center space-x-2">
          <Button variant="ghost" size="icon" aria-label="Dashboard">
            <LayoutGrid className="h-5 w-5" />
          </Button>
          <Button variant="ghost" size="icon" aria-label="Settings">
            <Settings className="h-5 w-5" />
          </Button>
        </nav> */}
      </div>
    </header>
  );
}
