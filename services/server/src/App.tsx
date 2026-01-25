import "./index.css";
import { apiClient } from "./client";
import { useEffect } from "react";
export function App() {
  useEffect(() => {
    apiClient.api.hello.get().then((value) => {
      console.log("Logs", value.data);
    });
  }, []);

  return (
    <div className="container mx-auto p-8 text-center relative z-10">
      <h1>Bunbase</h1>
    </div>
  );
}

export default App;
