import { BrowserRouter, Routes, Route } from "react-router-dom";
import { Layout } from "./components/Layout";
import { HomePage } from "./pages/HomePage";
import { About } from "./pages/About";
import { Docs } from "./pages/Docs";
import { DocView } from "./pages/DocView";

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<HomePage />} />
          <Route path="about" element={<About />} />
          <Route path="docs" element={<Docs />} />
          <Route path="docs/*" element={<DocView />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
