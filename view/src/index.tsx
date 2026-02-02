import { render } from "preact";

import "./style.css";

export function App() {
    return (
        <div>
            {" "}
            <h1 class="text-3xl font-bold underline">Weather Dashboard</h1>
            <input
                class="bg-gray-800 dark:text-white dark:border-gray-600 placeholder-gray-500 dark:focus:ring-blue-400"
                name="search"
                type="Text"
                placeholder="City or region..."
            ></input>
        </div>
    );
}

render(<App />, document.getElementById("app"));
