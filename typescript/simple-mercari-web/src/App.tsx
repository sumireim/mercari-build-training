import { useState } from 'react';
import './App.css';
import { ItemList } from '~/components/ItemList';
import { Listing } from '~/components/Listing';

function App() {
  // reload ItemList after Listing complete
  const [reload, setReload] = useState(true);
  return (
    <div>
      <header className="Title">
        <p>
          <b>Simple Mercari</b>
        </p>
      </header>
      <div className="ListingContainer">
        <Listing onListingCompleted={() => setReload(true)} />
      </div>
      <ItemList reload={reload} onLoadCompleted={() => setReload(false)} />
    </div>
  );
}
export default App;
