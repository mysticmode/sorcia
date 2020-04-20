function triggerCreateDropdown() {
  event.preventDefault()
  var dropdown = document.getElementsByClassName('create-new-dropdown active')
  if (dropdown.length > 0) {
      document.getElementsByClassName('create-new-dropdown')[0].classList.remove('active')
  } else {
      document.getElementsByClassName('create-new-dropdown')[0].classList.add('active')
  }
}