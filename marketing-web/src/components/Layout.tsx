import { Outlet, Link } from "react-router-dom";
import { getConsoleUrl } from "@/lib/config";

export function Layout() {
  const consoleUrl = getConsoleUrl();
  return (
    <div className="min-h-screen flex flex-col bg-base-100">
      <div className="navbar bg-base-200 shadow-lg">
        <div className="flex-1">
          <Link to="/" className="btn btn-ghost text-xl font-bold">
            BunBase
          </Link>
        </div>
        {/* Desktop menu */}
        <div className="flex-none hidden lg:block">
          <ul className="menu menu-horizontal px-1 gap-1">
            <li>
              <Link to="/">Home</Link>
            </li>
            <li>
              <Link to="/about">About</Link>
            </li>
            <li>
              <Link to="/docs">Docs</Link>
            </li>
            <li>
              <a
                href={consoleUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="btn btn-primary"
              >
                Open Console
              </a>
            </li>
          </ul>
        </div>
        {/* Mobile menu dropdown */}
        <div className="flex-none lg:hidden dropdown dropdown-end">
          <label tabIndex={0} className="btn btn-ghost btn-square">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              className="h-6 w-6"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M4 6h16M4 12h16M4 18h16"
              />
            </svg>
          </label>
          <ul
            tabIndex={0}
            className="menu dropdown-content bg-base-200 rounded-box z-50 mt-3 w-52 p-2 shadow-lg"
          >
            <li>
              <Link to="/">Home</Link>
            </li>
            <li>
              <Link to="/about">About</Link>
            </li>
            <li>
              <Link to="/docs">Docs</Link>
            </li>
            <li>
              <a
                href={consoleUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="btn btn-primary btn-sm"
              >
                Open Console
              </a>
            </li>
          </ul>
        </div>
      </div>
      <main className="flex-1">
        <Outlet />
      </main>
      <footer className="footer footer-center md:footer-center p-10 bg-base-200 text-base-content">
        <div className="grid grid-cols-1 gap-8 md:grid-cols-3 max-w-4xl mx-auto w-full">
          <div>
            <span className="footer-title">Product</span>
            <Link to="/docs" className="link link-hover">
              Docs
            </Link>
            <a
              href={consoleUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="link link-hover"
            >
              Console
            </a>
          </div>
          <div>
            <span className="footer-title">Company</span>
            <Link to="/about" className="link link-hover">
              About
            </Link>
          </div>
          <div>
            <span className="footer-title">Legal</span>
            <a href="#" className="link link-hover">
              Privacy
            </a>
            <a href="#" className="link link-hover">
              Terms
            </a>
          </div>
        </div>
        <aside className="mt-8">
          <p>© {new Date().getFullYear()} BunBase. All rights reserved.</p>
        </aside>
      </footer>
    </div>
  );
}
