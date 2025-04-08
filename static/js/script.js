document.addEventListener('DOMContentLoaded', () => {
    // Elements
    const movieSelect = document.getElementById('movieSelect');
    const showTimeSelect = document.getElementById('showTimeSelect');
    const seatsContainer = document.getElementById('seatsContainer');
    const bookingForm = document.getElementById('bookingForm');
    const bookButton = document.getElementById('bookButton');
    const selectedMovie = document.getElementById('selectedMovie');
    const selectedTime = document.getElementById('selectedTime');
    const selectedSeats = document.getElementById('selectedSeats');
    const totalPrice = document.getElementById('totalPrice');
    const logoutBtn = document.getElementById('logoutBtn');
    const loginBtn = document.getElementById('loginBtn');
    const registerBtn = document.getElementById('registerBtn');
    
    let currentMovie = null;
    let selectedSeatIds = new Set();
    let isAuthenticated = false;

    // Check authentication status
    checkAuth();

    // Load movies on page load
    fetchMovies();

    // Event Listeners
    movieSelect.addEventListener('change', handleMovieChange);
    showTimeSelect.addEventListener('change', handleShowTimeChange);
    bookingForm.addEventListener('submit', handleBooking);
    logoutBtn.addEventListener('click', handleLogout);

    // Check authentication status
    async function checkAuth() {
        try {
            const response = await fetch('/api/user');
            if (response.ok) {
                isAuthenticated = true;
                logoutBtn.classList.remove('d-none');
                loginBtn.classList.add('d-none');
                registerBtn.classList.add('d-none');
            } else {
                isAuthenticated = false;
                logoutBtn.classList.add('d-none');
                loginBtn.classList.remove('d-none');
                registerBtn.classList.remove('d-none');
            }
        } catch (error) {
            console.error('Error checking auth status:', error);
        }
    }

    // Handle logout
    async function handleLogout() {
        try {
            const response = await fetch('/auth/logout', {
                method: 'POST'
            });

            if (response.ok) {
                window.location.href = '/login';
            }
        } catch (error) {
            console.error('Error logging out:', error);
        }
    }

    // Fetch all movies
    async function fetchMovies() {
        try {
            const response = await fetch('/api/movies');
            const movies = await response.json();
            
            movieSelect.innerHTML = '<option value="">Choose a movie...</option>';
            movies.forEach(movie => {
                const option = document.createElement('option');
                option.value = movie.id;
                option.textContent = movie.title;
                option.dataset.price = movie.price;
                option.dataset.showTimes = JSON.stringify(movie.showTimes);
                movieSelect.appendChild(option);
            });
        } catch (error) {
            console.error('Error fetching movies:', error);
            alert('Failed to load movies. Please try again later.');
        }
    }

    // Handle movie selection
    function handleMovieChange() {
        const selectedOption = movieSelect.selectedOptions[0];
        if (!selectedOption.value) {
            showTimeSelect.disabled = true;
            showTimeSelect.innerHTML = '<option value="">Choose a show time...</option>';
            return;
        }

        currentMovie = {
            id: parseInt(selectedOption.value, 10),
            title: selectedOption.textContent,
            price: parseFloat(selectedOption.dataset.price),
            showTimes: JSON.parse(selectedOption.dataset.showTimes)
        };

        selectedMovie.textContent = currentMovie.title;
        showTimeSelect.disabled = false;
        
        // Populate show times
        showTimeSelect.innerHTML = '<option value="">Choose a show time...</option>';
        currentMovie.showTimes.forEach(time => {
            const option = document.createElement('option');
            option.value = time;
            option.textContent = time;
            showTimeSelect.appendChild(option);
        });

        updateBookingDetails();
    }

    // Handle show time selection
    async function handleShowTimeChange() {
        if (!showTimeSelect.value) {
            seatsContainer.innerHTML = '';
            return;
        }

        selectedTime.textContent = showTimeSelect.value;
        await loadSeats();
        updateBookingDetails();
    }

    // Load seats for selected show
    async function loadSeats() {
        try {
            const response = await fetch(`/api/seats?showId=${currentMovie.id}`);
            const seats = await response.json();
            
            seatsContainer.innerHTML = '';
            selectedSeatIds.clear();

            // Create 8x8 seat grid
            for (let i = 0; i < 8; i++) {
                for (let j = 0; j < 8; j++) {
                    const seatNumber = `${String.fromCharCode(65 + i)}${j + 1}`;
                    const seat = seats.find(s => s.number === seatNumber) || {
                        id: `${i}-${j}`,
                        number: seatNumber,
                        isBooked: false
                    };

                    const seatElement = document.createElement('div');
                    seatElement.className = `seat${seat.isBooked ? ' booked' : ''}`;
                    seatElement.textContent = seatNumber;
                    seatElement.dataset.id = seat.id;
                    seatElement.dataset.number = seatNumber;

                    if (!seat.isBooked) {
                        seatElement.addEventListener('click', () => toggleSeat(seatElement));
                    }

                    seatsContainer.appendChild(seatElement);
                }
            }
        } catch (error) {
            console.error('Error loading seats:', error);
            alert('Failed to load seats. Please try again later.');
        }
    }

    // Toggle seat selection
    function toggleSeat(seatElement) {
        const seatId = parseInt(seatElement.dataset.id, 10);
        const seatNumber = seatElement.dataset.number;

        if (selectedSeatIds.has(seatId)) {
            selectedSeatIds.delete(seatId);
            seatElement.classList.remove('selected');
        } else {
            selectedSeatIds.add(seatId);
            seatElement.classList.add('selected');
        }

        updateBookingDetails();
    }

    // Update booking details
    function updateBookingDetails() {
        const seatCount = selectedSeatIds.size;
        selectedSeats.textContent = seatCount ? `${seatCount} seats selected` : '-';
        totalPrice.textContent = seatCount ? `$${(currentMovie?.price * seatCount).toFixed(2)}` : '$0';
        bookButton.disabled = seatCount === 0 || !showTimeSelect.value;
    }

    // Handle booking submission
    async function handleBooking(event) {
        event.preventDefault();

        if (!isAuthenticated) {
            const loginRequiredModal = new bootstrap.Modal(document.getElementById('loginRequiredModal'));
            loginRequiredModal.show();
            return;
        }

        if (selectedSeatIds.size === 0 || !currentMovie || !showTimeSelect.value) {
            alert('Please select a movie, show time, and at least one seat');
            return;
        }

        try {
            const response = await fetch('/api/book', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    movieId: parseInt(currentMovie.id, 10),
                    showTime: showTimeSelect.value,
                    seatId: Array.from(selectedSeatIds)[0]
                })
            });

            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Booking failed');
            }

            // Show success modal
            const successModal = new bootstrap.Modal(document.getElementById('successModal'));
            successModal.show();

            // Reset form
            bookingForm.reset();
            selectedSeatIds.clear();
            await loadSeats();
            updateBookingDetails();

        } catch (error) {
            console.error('Error making booking:', error);
            alert(error.message || 'Failed to make booking. Please try again later.');
        }
    }
}); 