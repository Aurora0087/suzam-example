import { Link } from "@tanstack/react-router";
import { FaGithub } from "react-icons/fa";
import ThemeToggle from "./ThemeToggle";

export default function Header() {
  return (
    <header className="fixed w-full top-0 z-50 border-b-2 bg-background/50 backdrop-blur-sm">
      <nav className=' flex justify-between items-center uppercase'>
        <Link to="/" className=" px-6 w-full text-center h-12 grid place-content-center bg-transparent hover:bg-primary hover:text-primary-foreground transition-colors border-r-2">Identify</Link>
        <Link to="/songs" className=" px-6 w-full text-center h-12 grid place-content-center bg-transparent hover:bg-primary hover:text-primary-foreground transition-colors border-r-2">Collactions</Link>
        <Link to="/queues" className=" px-6 w-full text-center h-12 grid place-content-center bg-transparent hover:bg-primary hover:text-primary-foreground transition-colors border-r-2">Queue</Link>
        <a target="_blank" href="https://github.com/Aurora0087/suzam-example" className=" px-6 aspect-square text-center h-12 grid place-content-center bg-transparent hover:bg-primary hover:text-primary-foreground transition-colors border-r-2">
        <FaGithub />
        </a>
        <div className="aspect-square h-12 grid place-content-center">
          <ThemeToggle/>
        </div>
      </nav>
    </header>
  )
}
