import { useEffect } from "react";
import SpeedTest from "./components/SpeedTest";
import { DarkModeToggle } from "./components/DarkModeToggle";
import { initializeDarkMode } from "./utils/darkMode";

function App() {
  useEffect(() => {
    initializeDarkMode();
  }, []);

  return (
    <div className="min-h-screen bg-white dark:bg-gray-900">
      <div className="absolute top-4 right-4">
        <DarkModeToggle />
      </div>
      <SpeedTest />
    </div>
  );
}

export default App;
