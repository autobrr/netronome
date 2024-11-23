/*
 * Copyright (c) 2024, s0up and the autobrr contributors.
 * SPDX-License-Identifier: GPL-2.0-or-later
 */

import { useEffect } from "react";
import SpeedTest from "./components/SpeedTest";
// import { DarkModeToggle } from "./components/DarkModeToggle";
import { initializeDarkMode } from "./utils/darkMode";

function App() {
  useEffect(() => {
    initializeDarkMode();
  }, []);

  return (
    <div className="min-h-screen bg-white dark:bg-gray-900">
      <div className="absolute top-4 right-4">{/* <DarkModeToggle /> */}</div>
      <SpeedTest />
    </div>
  );
}

export default App;
